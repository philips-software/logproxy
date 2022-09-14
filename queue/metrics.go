package queue

type Metrics interface {
	IncProcessed()
	IncEnhancedTransactionID()
	IncEnhancedEncodedMessage()
}
