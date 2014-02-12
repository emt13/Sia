package swarm

import (
	"common"
	"crypto/sha256"
	"time"
)

type State interface {
	HandleTransaction(t common.Transaction)
	HandleBlock(b *Block)
}

func newBlockChain(Host string, Id string, StorageMapping map[string]interface{}) (b *BlockChain) {
	b = new(BlockChain)
	b.Id = Id
	b.StorageMapping = StorageMapping
	b.outgoingTransactions = make(chan common.NetworkObject)
	return
}

func NewBlockChain(Host string, Id string, StorageMapping map[string]interface{}) (b *BlockChain) {
	b = newBlockChain(Host, Id, StorageMapping)
	b.state = NewStateSwarmInformed(b)
	return
}

func JoinBlockChain(Host string, Id string) (b *BlockChain) {
	b = newBlockChain(Host, Id, make(map[string]interface{}))
	//b.state = NewStateSwarmJoin(b)
	return
}

//List of states
// SwarmInformed - Swarm member shave been told to join swarm
// SwarmConnected - Swarm Members have succesfully formed a block
// SwarmLive - Swarm has sent a signal to the parent blockchain saying it is
//             alive and is in the steady state
// SwarmJoin - We are joining an already alive swarm
// SwarmDied - The swarm has died, terminate

type StateSwarmInformed struct {
	//Map of hosts seen to number of times they have failed to generate a block
	//Used for both host alive tracking & host block generation tracking
	hostsseen map[string]int

	// How many times we have broadcast that we are alive, we use a two stage
	// process where we broadcast, and then broadcast again when we have seen
	// enough nodes up to form a majority
	broadcastcount uint

	// This state has two phases, the learning phase where it listens for new
	// hosts and then the commit stage where it listens for a block that
	// is correct according to its knowledge and then votes for it.
	learning bool

	chain    *BlockChain
	blockgen <-chan time.Time
	block    *Block
}

func NewStateSwarmInformed(chain *BlockChain) (s *StateSwarmInformed) {
	s = new(StateSwarmInformed)
	s.chain = chain
	s.blockgen = time.Tick(5 * time.Second)

	s.learning = true

	go s.broadcastLife()
	go s.checkBlockGen()
	return
}

func (s *StateSwarmInformed) blockCompiler() (compiler string) {

	hosts := make([]string, 0, len(s.hostsseen))

	//Pull all hosts who we haven't seen skipping a block
	for host, skipped := range s.hostsseen {
		if skipped != 0 {
			continue
		}
		hosts = append(hosts, host)
	}

	//Check if we should be the block generator
	compiler = common.RendezvousHash(sha256.New(), hosts, s.chain.Host)
	return
}

func (s *StateSwarmInformed) checkBlockGen() {

	var compiler string

	for _ = range s.blockgen {

		if s.learning {
			s.learning = false
			continue
		}

		if len(s.chain.BlockHistory) == 0 {
			s.hostsseen[compiler] += 1
		}

		//Dont't try to generate a block if we don't have a majority of hosts
		if len(s.hostsseen) < 128 {
			continue
			//Should actually switch to state swarmdied / join
		}

		compiler = s.blockCompiler()

		if compiler == s.chain.Host {

			id, err := common.RandomString(8)
			if err != nil {
				panic("checkBlockGenRandom" + err.Error())
			}
			b := &Block{id, s.chain.Id, s.chain.Host, nil, nil, nil}
			b.StorageMapping = make(map[string]interface{})
			for host, _ := range s.hostsseen {
				b.StorageMapping[host] = nil
			}

			s.chain.outgoingTransactions <- common.BlockNetworkObject(b)
		}
	}
}

func (s *StateSwarmInformed) broadcastLife() {
	s.broadcastcount += 1
	s.chain.outgoingTransactions <- common.TransactionNetworkObject(
		NewNodeAlive(s.chain.Host, s.chain.Id))
}

func (s *StateSwarmInformed) HandleTransaction(t common.Transaction) {
	switch n := t.(type) {
	case *NodeAlive:
		if !s.learning {
			return
		}

		s.hostsseen[n.Node] = 0
		// Resend hostsseen count once you have seen a majority of hosts
		if len(s.hostsseen) > 128 && s.broadcastcount < 2 {
			s.broadcastLife()
		}
	default:
		return
	}
}

func (s *StateSwarmInformed) HandleBlock(b *Block) {

	// If the learning timeout hasn't expired, don't accept blocks
	if s.learning {
		return
	}

	// All blocks in this state should be generated by the ideal host
	if b.Compiler != s.blockCompiler() {
		return
	}

	// We are looking for a block to generate a heartbeat for
	if len(s.chain.BlockHistory) == 0 {
		s.chain.AddBlock(b)

		if _, ok := b.StorageMapping[s.chain.Host]; ok {
			//Generate heartbeat for block
		}
	}

	// We're looking for the block with heartbeats to figure out if we're in
	// it
	if len(s.chain.BlockHistory) == 1 {
		if _, ok := b.StorageMapping[s.chain.Host]; ok {
			//If we're in the block switch to signal mode
			//s.chain.state = NewStateSwarmConnected()
		} else {
			//Join the swarm
			//s.chain.state = NewStateSwarmJoin()

		}
	}

}
