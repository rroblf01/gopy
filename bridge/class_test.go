//go:build cgo

package bridge

import "testing"

// TestPydanticModelFromGo defines a Pydantic model entirely from Go via
// MakeClass (inheriting pydantic.BaseModel with annotated fields), then
// instantiates and validates it through Pydantic's machinery. Proves Go can
// declare framework model classes that the framework treats as native.
func TestPydanticModelFromGo(t *testing.T) {
	pydantic, err := Import("pydantic")
	if err != nil {
		t.Skipf("pydantic not available: %v", err)
	}
	defer pydantic.DecRef()
	baseModel, err := pydantic.Attr("BaseModel")
	if err != nil {
		t.Fatalf("pydantic.BaseModel: %v", err)
	}
	defer baseModel.DecRef()

	// class User(BaseModel): name: str; age: int
	user, err := MakeClass("User", []*Object{baseModel},
		[]Field{{Name: "name", Annotation: "str"}, {Name: "age", Annotation: "int"}})
	if err != nil {
		t.Fatalf("MakeClass(User): %v", err)
	}
	defer user.DecRef()

	// Instantiate with validation/coercion: age "3" (str) coerces to int 3.
	u, err := user.CallKw(nil, map[string]any{"name": "ana", "age": "3"})
	if err != nil {
		t.Fatalf("User(name=ana, age='3'): %v", err)
	}
	defer u.DecRef()

	name, err := u.Attr("name")
	if err != nil {
		t.Fatalf("u.name: %v", err)
	}
	defer name.DecRef()
	nv, _ := name.Go()
	if nv != "ana" {
		t.Fatalf("u.name = %v, want ana", nv)
	}

	age, err := u.Attr("age")
	if err != nil {
		t.Fatalf("u.age: %v", err)
	}
	defer age.DecRef()
	av, _ := age.Go()
	if av != int64(3) {
		t.Fatalf("u.age = %v, want 3 (coerced from '3')", av)
	}

	// Validation failure: age "notanumber" must raise (Pydantic ValidationError).
	if _, err := user.CallKw(nil, map[string]any{"name": "x", "age": "notanumber"}); err == nil {
		t.Fatal("expected validation error for bad age, got nil")
	}
}
