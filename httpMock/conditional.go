package httpMock

import (
	"net/http"
	"gopkg.in/xmlpath.v2"
)

type RequestPredicate func(*http.Request) bool

var RequestPredicateTrue RequestPredicate = func(r *http.Request) bool {
	return true
}

type RequestKeySupplier func(*http.Request) interface{}

var RequestKeyIdentity RequestKeySupplier = func(r *http.Request) interface{} {
	return r
}

// Uses XPATH expression to extract a string from the body of the request.
func ExtractXPathString(xpath string) RequestKeySupplier {
	path := xmlpath.MustCompile(xpath)
	return func(r *http.Request) interface{} {
		str := ""
		root, err := xmlpath.Parse(r.Body)
		if err == nil {
			str, _ = path.String(root)
		}
		return str
	}
}

type RequestKeyPredicate func(interface{}) bool

func RequestKeyStringEquals(str string) RequestKeyPredicate {
	return func(key interface{}) bool {
		return key.(string) == str;
	}
}

type when struct {
	predicate     RequestPredicate
	trueResponse  http.Handler
	falseResponse http.Handler
}

func When(predicate RequestPredicate, trueResponseBuilder func(), falseResponseBuilder func()) {

	outerMockMethod := currentMockMethod
	Method(outerMockMethod.method, trueResponseBuilder)
	trueMockMethod := currentMock.methods[outerMockMethod.method]
	Method(outerMockMethod.method, falseResponseBuilder)
	falseMockMethod := currentMock.methods[outerMockMethod.method]

	currentMockMethodHandler = &when{predicate, trueMockMethod, falseMockMethod}
}

func (wh *when) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	if wh.predicate(request) {
		wh.trueResponse.ServeHTTP(w, request)
	} else {
		wh.falseResponse.ServeHTTP(w, request)
	}
}

type switchCase struct {
	predicate RequestKeyPredicate
	response  http.Handler
}

type switchCaseSet struct {
	keySupplier    RequestKeySupplier
	switchCases    []*switchCase
	defaultHandler http.Handler
}

func (scs *switchCaseSet) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	key := scs.keySupplier(request)
	for _, sc := range scs.switchCases {
		if sc.predicate(key) {
			sc.response.ServeHTTP(w, request)
			return
		}
	}
	scs.defaultHandler.ServeHTTP(w, request)
}

var currentSwitch *switchCaseSet

func Switch(keySupplier RequestKeySupplier, cases func()) {
	currentSwitch = &switchCaseSet{
		keySupplier:    keySupplier,
		switchCases:    make([]*switchCase, 0, 10),
		defaultHandler: NotFound,
	}
	cases()
	currentMockMethodHandler = currentSwitch
}

func Case(predicate RequestKeyPredicate, responseBuilder func()) {
	if currentMockMethod == nil {
		panic("Switch must be inside a method.")
	}
	outerMockMethod := currentMockMethod
	Method(outerMockMethod.method, responseBuilder)
	responseMockMethod := currentMock.methods[outerMockMethod.method]
	if predicate != nil {
		currentSwitch.switchCases = append(currentSwitch.switchCases, &switchCase{predicate, responseMockMethod})
	} else {
		currentSwitch.defaultHandler = responseMockMethod
	}
	currentMockMethod = outerMockMethod
}

func Default(responseBuilder func()) {
	Case(nil, responseBuilder)
}
