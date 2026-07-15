package good

func helperUsedOnlyInTests() {
	go named() // _test.go files are exempt; no diagnostic expected
}
