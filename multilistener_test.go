package multilistener

import (
	"io"
	"net"
	"slices"
	"sync"
	"testing"
)

// TestMultiListen tests the initial listener.
func TestMultiListen(t *testing.T) {
	m, err := Listen(map[string][]string{
		"tcp":  {"127.0.0.1:8080"},
		"tcp6": {"[::1]:8080"},
	})

	if err != nil {
		t.Error("error when listening on valid addresses", err)
	}

	t.Cleanup(func() {
		m.Close()
	})
}

// TestMultiListenAddr tests listening and getting the address of listeners.
func TestMultiListenAddr(t *testing.T) {
	m, err := Listen(map[string][]string{
		"tcp":  {"127.0.0.1:8080"},
		"tcp6": {"[::1]:8080"},
	})

	if err != nil {
		t.Error("error when listening on valid addresses", err)
	}

	network := m.Addr().Network()
	address := m.Addr().String()

	if network != "tcp;tcp" {
		t.Error("network should be the listener networks separated by a semicolon", network)
	}

	if address != "127.0.0.1:8080;[::1]:8080" && address != "[::1]:8080;127.0.0.1:8080" {
		t.Error("listen addresses should be the listener addresses separated by a semicolon", address)
	}

	t.Cleanup(func() {
		m.Close()
	})
}

// TestMultiListenAddresses listens on multiple interfaces and gets a list of listener addresses.
func TestMultiListenAddresses(t *testing.T) {
	m, err := Listen(map[string][]string{
		"tcp":  {"127.0.0.1:8080"},
		"tcp6": {"[::1]:8080"},
	})

	i0, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	i1, _ := net.ResolveTCPAddr("tcp6", "[::1]:8080")

	if err != nil {
		t.Error("error when listening on valid addresses", err)
	}

	if a, ok := m.(*MultiListener); ok {
		for _, addr := range a.Addresses() {
			if !slices.Contains([]string{i0.Network(), i1.Network()}, addr.Network()) || !slices.Contains([]string{i0.String(), i1.String()}, addr.String()) {
				t.Error("addresses do not match")
			}
		}
	} else {
		t.Error("not a multilistener")
	}

	t.Cleanup(func() {
		m.Close()
	})
}

// TestMultiListenMultipleClose tests listening and closing multiple times.
func TestMultiListenMultipleClose(t *testing.T) {
	m, err := Listen(map[string][]string{
		"tcp":  {"127.0.0.1:8080"},
		"tcp6": {"[::1]:8080"},
	})

	if err != nil {
		t.Error("error when listening on valid addresses", err)
	}

	err = m.Close()
	if err != nil {
		t.Error("error when closing", err)
	}

	err = m.Close()
	if err == nil {
		t.Error("listener should already be closed", err)
	}

	t.Cleanup(func() {
		m.Close()
	})
}

// TestMultiListenCloseError tests how a close error bubbles up.
func TestMultiListenCloseError(t *testing.T) {
	m, err := Listen(map[string][]string{
		"tcp":  {"127.0.0.1:8080"},
		"tcp6": {"[::1]:8080"},
	})

	if err != nil {
		t.Error("error when listening on valid addresses", err)
	}

	if a, ok := m.(*MultiListener); ok {
		for _, v := range a.listeners {
			err = v.Close()
			if err != nil {
				t.Error("first listener should be okay", err)
			}
			break
		}
	} else {
		t.Error("not a multilistener")
	}

	err = m.Close()
	if err == nil {
		t.Error("should error since first listener is closed")
	}

	t.Cleanup(func() {
		m.Close()
	})
}

// TestMultiListenAccept tests multiple listeners with a single accept routine.
func TestMultiListenAccept(t *testing.T) {
	listeners := map[string][]string{
		"tcp":  {"127.0.0.1:8080"},
		"tcp6": {"[::1]:8080"},
	}

	m, err := Listen(listeners)

	if err != nil {
		t.Error("error when listening on valid addresses", err)
	}

	var wg sync.WaitGroup

	wg.Add(2)

	msg := "Hello world!"

	go func() {
		for i := 0; i < 2; i++ {
			c, err := m.Accept()
			if err != nil {
				t.Error("error accepting listener", err)
			}

			n, err := io.ReadAll(c)
			if err != nil {
				t.Error("error reading from listener", err)
			}

			if string(n) != msg {
				t.Error("read data is not same as sent", string(n))
			}

			err = c.Close()
			if err != nil {
				t.Error("error closing connection", err)
			}
		}
		wg.Done()
	}()

	go func() {
		for network, addresses := range listeners {
			for _, address := range addresses {
				c, err := net.Dial(network, address)
				if err != nil {
					t.Error("error connecting to listener", err)
				}

				_, err = c.Write([]byte(msg))
				if err != nil {
					t.Error("error writing to listener", err)
				}

				err = c.Close()
				if err != nil {
					t.Error("error closing listener", err)
				}
			}
		}
		wg.Done()
	}()

	wg.Wait()

	err = m.Close()
	if err != nil {
		t.Error("should not error on close", err)
	}

	t.Cleanup(func() {
		m.Close()
	})
}

// TestMultiListenAcceptAndClose tests what happens when a close occurs during an accept.
func TestMultiListenAcceptAndClose(t *testing.T) {
	listeners := map[string][]string{
		"tcp":  {"127.0.0.1:8080"},
		"tcp6": {"[::1]:8080"},
	}

	m, err := Listen(listeners)

	if err != nil {
		t.Error("error when listening on valid addresses", err)
	}

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		for i := 0; i < 2; i++ {
			_, err := m.Accept()
			if err == nil {
				t.Error("error should be observed because connection is closed", err)
			}
		}
		wg.Done()
	}()

	err = m.Close()
	if err != nil {
		t.Error("should not error on close", err)
	}

	wg.Wait()

	t.Cleanup(func() {
		m.Close()
	})
}

// TestErrorOnListen tests that a error occurs when listening if they are invalid.
func TestErrorOnListen(t *testing.T) {
	_, err := Listen(map[string][]string{
		"foobar": {"baz"},
	})

	if err == nil {
		t.Error("no error when using invalid listen type")
	}
}
