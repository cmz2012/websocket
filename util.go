package websocket

func isControlFrame(OpCode byte) bool {
	return OpCode != PayloadTypeText && OpCode != PayloadTypeBinary
}
