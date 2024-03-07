// multilistener implements a net.Listener that can accept multiple networks and addresses
package multilistener

import (
	"errors"
	"net"
	"strings"
	"sync"
)

var ErrClosed = errors.New("listener is already closed")

type chanMsg struct {
	conn net.Conn
	err  error
}

// MultiListener is the main multilistener struct.
type MultiListener struct {
	mut       *sync.RWMutex
	listeners map[net.Addr]net.Listener
	accept    chan chanMsg
	stop      chan struct{}
}

// Network implements net.Addr.
func (m *MultiListener) Network() string {
	m.mut.RLock()
	defer m.mut.RUnlock()

	a := []string{}
	for addr := range m.listeners {
		a = append(a, addr.Network())
	}
	return strings.Join(a, ";")
}

// String implements net.Addr.
func (m *MultiListener) String() string {
	m.mut.RLock()
	defer m.mut.RUnlock()

	a := []string{}
	for addr := range m.listeners {
		a = append(a, addr.String())
	}
	return strings.Join(a, ";")
}

// Addresses returns a slice of addresses. This is not ordered.
func (m *MultiListener) Addresses() []net.Addr {
	m.mut.RLock()
	defer m.mut.RUnlock()

	a := []net.Addr{}
	for addr := range m.listeners {
		a = append(a, addr)
	}
	return a
}

// Accept implements net.Listener.
func (m *MultiListener) Accept() (net.Conn, error) {
	select {
	case <-m.stop:
		return nil, ErrClosed
	case res := <-m.accept:
		return res.conn, res.err
	}
}

// Addr implements net.Listener.
func (m *MultiListener) Addr() net.Addr {
	return m
}

// Close implements net.Listener.
func (m *MultiListener) Close() error {
	m.mut.Lock()
	defer m.mut.Unlock()

	select {
	case <-m.stop:
		return ErrClosed
	default:
		closeErrs := []error{}

		for _, l := range m.listeners {
			err := l.Close()
			if err != nil {
				closeErrs = append(closeErrs, err)
			}
		}

		close(m.stop)

		return errors.Join(closeErrs...)
	}
}

// Listen listens on multiple network->[]address pairs as defined in the map.
func Listen(listeners map[string][]string) (net.Listener, error) {
	m := &MultiListener{
		mut:       &sync.RWMutex{},
		listeners: map[net.Addr]net.Listener{},
		accept:    make(chan chanMsg),
		stop:      make(chan struct{}),
	}

	m.mut.Lock()
	defer m.mut.Unlock()

	for network, addresses := range listeners {
		for _, address := range addresses {
			nL, err := net.Listen(network, address)
			if err != nil {
				return nil, err
			}

			m.listeners[nL.Addr()] = nL
		}
	}

	for _, l := range m.listeners {
		go func(l net.Listener) {
			for {
				c, e := l.Accept()
				msg := chanMsg{conn: c, err: e}
				select {
				case <-m.stop:
					return
				case m.accept <- msg:
					continue
				}
			}
		}(l)
	}

	return m, nil
}

var _ net.Listener = &MultiListener{}
var _ net.Addr = &MultiListener{}
