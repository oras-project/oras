/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package progress

import (
	"os"
	"sync"
	"time"

	"oras.land/oras/cmd/oras/internal/display/console"
)

const BUFFER_SIZE = 20

// Status is print message channel
type Status chan<- *status

// Manager is progress view master
type Manager interface {
	Add() Status
	StopAndWait()
}

const (
	bufFlushDuration = 100 * time.Millisecond
)

type manager struct {
	statuses   []*status
	rwLock     sync.RWMutex
	renderTick *time.Ticker
	c          *console.Console
	updating   sync.WaitGroup
	mu         sync.Mutex
	close      sync.Once
	// done used to stop render routine
	// doneDone used to mark render routine stopped
	done     chan struct{}
	doneDone chan struct{}
}

// NewManager initialized a new progress manager.
func NewManager(f *os.File) (Manager, error) {
	var m manager
	var err error

	m.c, err = console.GetConsole(f)
	if err != nil {
		return nil, err
	}
	m.done = make(chan struct{})
	m.doneDone = make(chan struct{})
	m.renderTick = time.NewTicker(bufFlushDuration)
	m.start()
	return &m, nil
}

func (m *manager) start() {
	m.renderTick.Reset(bufFlushDuration)
	m.c.Save()
	go func() {
		for {
			m.render()
			select {
			case <-m.done:
				close(m.doneDone)
				return
			case <-m.renderTick.C:
			}
		}
	}()
}

func (m *manager) render() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// todo: update size in another routine
	width, height := m.c.Size()
	len := len(m.statuses) * 2
	offset := 0
	if len > height {
		// skip statuses that cannot be rendered
		offset = len - height
	}

	for ; offset < len; offset += 2 {
		m.rwLock.RLock()
		status, progress := m.statuses[offset/2].String(width)
		m.rwLock.RUnlock()
		m.c.OutputTo(uint(len-offset), status)
		m.c.OutputTo(uint(len-offset-1), progress)
	}
}

// Add appends a new status with 2-line space for rendering.
func (m *manager) Add() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := len(m.statuses)
	m.statuses = append(m.statuses, nil)
	defer m.c.NewRow()
	defer m.c.NewRow()
	return m.newStatus(id)
}

func (m *manager) newStatus(id int) Status {
	ch := make(chan *status, BUFFER_SIZE)
	m.updating.Add(1)
	go m.update(ch, id)
	return ch
}

func (m *manager) update(ch chan *status, id int) {
	defer m.updating.Done()
	for s := range ch {
		m.rwLock.Lock()
		m.statuses[id] = m.statuses[id].Update(s)
		m.rwLock.Unlock()
	}
}

// StopAndWait stops all status and waits for updating and rendering.
func (m *manager) StopAndWait() {
	// 1. stop periodic render
	m.renderTick.Stop()
	close(m.done)
	defer m.close.Do(func() {
		// 4. restore cursor, mark done
		m.c.Restore()
	})
	// 2. wait for all model update done
	m.updating.Wait()
	// 3. render last model
	<-m.doneDone
	m.render()
}
