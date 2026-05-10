package finder

// Subject identifies what kind of entity a query is asking about. It drives
// output grouping in the renderer.
type Subject string

const (
	SubjectType Subject = "type"
	SubjectFunc Subject = "func"
)

// InferSubject decides whether a query is asking about types or functions.
// The rule:
//  1. --has-method or --implements => type
//  2. --kind in {struct, interface, alias, named} => type
//  3. otherwise => func (covers --kind=func/method, --returns, --takes,
//     --has-receiver, and any combination of those)
func InferSubject(f Filter) Subject {
	if f.HasMethod != "" || f.Implements != "" {
		return SubjectType
	}
	switch f.Kind {
	case "struct", "interface", "alias", "named":
		return SubjectType
	}
	return SubjectFunc
}
