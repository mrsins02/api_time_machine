package events

// Event contracts for cross-service integration.

const (
	UserCreated    = "UserCreated"
	UserUpdated    = "UserUpdated"
	UserDeleted    = "UserDeleted"
	EmailChanged   = "EmailChanged"
	ProductCreated = "ProductCreated"
	ProductUpdated = "ProductUpdated"
	OrderCreated   = "OrderCreated"
	OrderCancelled = "OrderCancelled"
)

const NATSTopicDefault = "atm.events"
