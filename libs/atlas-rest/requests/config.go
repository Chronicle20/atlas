package requests

import "time"

type configuration struct {
	retries          int
	timeout          time.Duration
	headerDecorators []HeaderDecorator
}

type Configurator func(c *configuration)

//goland:noinspection GoUnusedExportedFunction
func SetRetries(amount int) Configurator {
	return func(c *configuration) {
		c.retries = amount
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetTimeout(d time.Duration) Configurator {
	return func(c *configuration) {
		c.timeout = d
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetHeaderDecorator(hd HeaderDecorator) Configurator {
	return func(c *configuration) {
		c.headerDecorators = []HeaderDecorator{hd}
	}
}

//goland:noinspection GoUnusedExportedFunction
func AddHeaderDecorator(hd HeaderDecorator) Configurator {
	return func(c *configuration) {
		c.headerDecorators = append(c.headerDecorators, hd)
	}
}
