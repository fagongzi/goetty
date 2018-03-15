package goetty

// Middleware goetty middleware
type Middleware interface {
	PreRead(conn IOSession) (bool, interface{}, error)
	PostRead(msg interface{}, conn IOSession) (bool, interface{}, error)

	// PreWrite middl1 PostWrite -> middle2 PostWrite -> middleN PostWrite -> do write
	PreWrite(msg interface{}, conn IOSession) (bool, interface{}, error)
	// PostWrite do write ->  middleN PreWrite -> middle2 PreWrite -> middle1 PreWrite
	PostWrite(msg interface{}, conn IOSession) (bool, error)

	Closed(conn IOSession)
	Connected(conn IOSession)
}

// BaseMiddleware defined default reutrn value
type BaseMiddleware struct {
}

// PostWrite default reutrn value
func (sm *BaseMiddleware) PostWrite(msg interface{}, conn IOSession) (bool, error) {
	return true, nil
}

// PreWrite default reutrn value
func (sm *BaseMiddleware) PreWrite(msg interface{}, conn IOSession) (bool, interface{}, error) {
	return true, msg, nil
}

// PreRead default reutrn value
func (sm *BaseMiddleware) PreRead(conn IOSession) (bool, interface{}, error) {
	return true, nil, nil
}

// PostRead default reutrn value
func (sm *BaseMiddleware) PostRead(msg interface{}, conn IOSession) (bool, interface{}, error) {
	return false, true, nil
}

// Closed default option
func (sm *BaseMiddleware) Closed(conn IOSession) {

}

// Connected default option
func (sm *BaseMiddleware) Connected(conn IOSession) {

}
