package parallel

import (
	"context"
	"sync"

	"shop/pkg/recommend/offline/train/util"

	"github.com/samber/lo"
)

const chanSize = 1024

/* Parallel Schedulers */

func Parallel(ctx context.Context, nJobs, nWorkers int, worker func(workerId, jobId int) error) error {
	if nWorkers <= 1 {
		for i := range nJobs {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := worker(0, i); err != nil {
				return err
			}
		}
	} else {
		c := make(chan int, chanSize)
		// producer
		go func() {
			defer close(c)
			for i := range nJobs {
				select {
				case <-ctx.Done():
					return
				case c <- i:
				}
			}
		}()
		// consumer
		var wg sync.WaitGroup
		errs := make([]error, nJobs)
		for j := range nWorkers {
			// start workers
			workerId := j
			wg.Go(func() {
				defer util.CheckPanic()
				for {
					select {
					case <-ctx.Done():
						return
					case jobId, ok := <-c:
						if !ok {
							return
						}
						if err := ctx.Err(); err != nil {
							errs[jobId] = err
							return
						}
						// run job
						if err := worker(workerId, jobId); err != nil {
							errs[jobId] = err
							return
						}
					}
				}
			})
		}
		wg.Wait()
		// check errors
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
	}
	return ctx.Err()
}

func For(ctx context.Context, nJobs, nWorkers int, worker func(int)) error {
	if nWorkers <= 1 {
		for i := range nJobs {
			if err := ctx.Err(); err != nil {
				return err
			}
			worker(i)
		}
	} else {
		c := make(chan int, chanSize)
		// producer
		go func() {
			defer close(c)
			for i := range nJobs {
				select {
				case <-ctx.Done():
					return
				case c <- i:
				}
			}
		}()
		// consumer
		var wg sync.WaitGroup
		for range nWorkers {
			// start workers
			wg.Go(func() {
				for {
					select {
					case <-ctx.Done():
						return
					case jobId, ok := <-c:
						if !ok {
							return
						}
						if err := ctx.Err(); err != nil {
							return
						}
						worker(jobId)
					}
				}
			})
		}
		wg.Wait()
	}
	return ctx.Err()
}

func ForEach[T any](ctx context.Context, a []T, nWorkers int, worker func(int, T)) error {
	if nWorkers <= 1 {
		for i, v := range a {
			if err := ctx.Err(); err != nil {
				return err
			}
			worker(i, v)
		}
	} else {
		c := make(chan lo.Tuple2[int, T], chanSize)
		// producer
		go func() {
			defer close(c)
			for i, v := range a {
				select {
				case <-ctx.Done():
					return
				case c <- lo.Tuple2[int, T]{A: i, B: v}:
				}
			}
		}()
		// consumer
		var wg sync.WaitGroup
		for range nWorkers {
			// start workers
			wg.Go(func() {
				for {
					select {
					case <-ctx.Done():
						return
					case job, ok := <-c:
						if !ok {
							return
						}
						if err := ctx.Err(); err != nil {
							return
						}
						worker(job.A, job.B)
					}
				}
			})
		}
		wg.Wait()
	}
	return ctx.Err()
}

func Split[T any](a []T, n int) [][]T {
	if len(a) == 0 {
		return nil
	}
	if n > len(a) {
		n = len(a)
	}
	minChunkSize := len(a) / n
	maxChunkNum := len(a) % n
	chunks := make([][]T, n)
	for i, j := 0, 0; i < n; i++ {
		chunkSize := minChunkSize
		if i < maxChunkNum {
			chunkSize++
		}
		chunks[i] = a[j : j+chunkSize]
		j += chunkSize
	}
	return chunks
}

type Context struct {
	sem         chan struct{}
	detachedSem chan struct{}
	detached    bool
}

func (ctx *Context) Detach() {
	if ctx == nil || ctx.detached {
		return
	}
	ctx.detachedSem <- struct{}{}
	ctx.detached = true
	<-ctx.sem
}

func (ctx *Context) Attach() {
	if ctx == nil || !ctx.detached {
		return
	}
	ctx.detached = false
	<-ctx.detachedSem
	ctx.sem <- struct{}{}
}

func Detachable(ctx context.Context, nJobs, nWorkers, nMaxDetached int, worker func(*Context, int)) error {
	sem := make(chan struct{}, nWorkers)
	detachedSem := make(chan struct{}, nMaxDetached)
	var wg sync.WaitGroup
	for i := range nJobs {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Go(func() {
			if ctx.Err() != nil {
				<-sem
				return
			}
			c := &Context{sem: sem, detachedSem: detachedSem}
			worker(c, i)
			if c.detached {
				<-c.detachedSem
			} else {
				<-sem
			}
		})
	}
	wg.Wait()
	return ctx.Err()
}
