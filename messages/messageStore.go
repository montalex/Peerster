package messages

/*PastMessages represents the history of messages recieved from each node
MessagesList: the list of messages recieved from each node
*/
type PastMessages struct {
	MessagesList map[string][]*RumorMessage
}
