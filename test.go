package zlutils

type Con struct {
	Times int
	Done  chan int
}

func NewCon(times int) *Con {
	return &Con{
		Times: times,
		Done:  make(chan int, times),
	}
}

func (con *Con) Do(f func(c int)) *Con {
	for c := 0; c < con.Times; c++ { //并发times个线程
		go func(c int) {
			f(c)
			con.Done <- c
		}(c)
	}
	return con
}
func (con *Con) Wait() *Con { //等待所有线程完毕
	for c := 0; c < con.Times; c++ {
		<-con.Done
	}
	return con
}
