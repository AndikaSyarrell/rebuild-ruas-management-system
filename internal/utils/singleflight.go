package utils

import "sync"

// SingleflightGroup memastikan hanya SATU eksekusi fn() yang berjalan untuk key
// yang sama pada satu waktu; goroutine lain dengan key sama menunggu dan
// menerima hasil yang sama, alih-alih ikut menjalankan pekerjaan berat yang
// sama secara duplikat (mis. 5 request export dengan filter identik yang
// datang bersamaan saat cache miss -> cukup 1 yang benar-benar query+generate file).
type SingleflightGroup struct {
	mu    sync.Mutex
	calls map[string]*sfCall
}

type sfCall struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

func (g *SingleflightGroup) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.calls == nil {
		g.calls = make(map[string]*sfCall)
	}
	if c, ok := g.calls[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(sfCall)
	c.wg.Add(1)
	g.calls[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()

	return c.val, c.err
}
