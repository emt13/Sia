package quorum

import (
	"errors"
)

func (s *State) AddWallet(id string, bal uint64) (err error) {
	if bal == 0 {
		return errors.New("Cannot add balance of 0!")
	}
	elem, ok := s.wallets[id]
	if ok {
		return errors.New("This wallet already exists!")
	} else {
		elem = bal
		s.wallets[id] = elem
	}
	return nil
}

func (s *State) MoveBal(src string, dest string, amt uint64) (err error) {
	//check to make sure the wallets exist
	elem, ok := s.wallets[dest]
	if ok {
		return errors.New("Destination wallet does not exist!")
	}
	elem, ok = s.wallets[src]
	if ok {
		return errors.New("Source wallet does not exist!")
	}

	//check balance editting
	tmp := elem - amt
	if tmp < 0 {
		return errors.New("Source wallet does not have enough coins!")
	} else if tmp == 0 {
		delete(s.wallets, src)
	} else {
		s.wallets[src] = tmp
	}

	//change balance in destination
	s.wallets[dest] += tmp

	return nil
}
