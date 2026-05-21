package scanner

type Result struct {
	Root   string
	Status string
}

func Analyze(root string) Result {
	return Result{
		Root:   root,
		Status: "scanner placeholder initialized",
	}
}
