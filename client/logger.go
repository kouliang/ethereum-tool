package client

type logger interface {
	Println(a ...interface{})
	Printf(format string, a ...interface{})
}
