package gae

import "testing"

func runtest(t *testing.T, name, exp, act string) {
	if exp != act {
		t.Errorf("expect '%v' to return\n'%v'; got\n'%v'", name, exp, act)
	}
}

func TestErrors2(t *testing.T) {
	ea1 := DuplicateError{}
	runtest(t, "DuplicateError.Error - basic", "Duplicate value", ea1.Error())
	ea2 := DuplicateError{Name: "email"}
	runtest(t, "DuplicateError.Error - with name", "Duplicate value for email", ea2.Error())
	ea3 := DuplicateError{Name: "email", Msg: "must be unique"}
	runtest(t, "DuplicateError.Error - with name and msg", "Duplicate value for email - must be unique", ea3.Error())
	if !IsDuplicateError(ea3) {
		t.Errorf("expect IsDuplicateError to return true; got false")
	}

	eb1 := InsufficientError{}
	runtest(t, "InsufficientError.Error - basic", "Insufficient value", eb1.Error())
	eb2 := InsufficientError{Name: "email"}
	runtest(t, "InsufficientError.Error - with name", "Insufficient value for email", eb2.Error())
	eb3 := InsufficientError{Name: "email", Msg: "must be more than 1"}
	runtest(t, "InsufficientError.Error - with name and msg", "Insufficient value for email - must be more than 1", eb3.Error())
	if !IsInsufficientError(eb3) {
		t.Errorf("expect IsInsufficientError to return true; got false")
	}

	ec1 := InvalidError{}
	runtest(t, "InvalidError.Error - basic", "Invalid value ()", ec1.Error())
	ec2 := InvalidError{"email"}
	runtest(t, "InvalidError.Error - with msg", "Invalid value (email)", ec2.Error())
	if !IsInvalidError(ec2) {
		t.Errorf("expect IsInvalidError to return true; got false")
	}

	ed1 := MismatchError{}
	runtest(t, "MismatchError.Error - basic", "Mismatched values", ed1.Error())
	ed2 := MismatchError{"IDs are different"}
	runtest(t, "MismatchError.Error - with msg", "Mismatched values - IDs are different", ed2.Error())
	if !IsMismatchError(ed2) {
		t.Errorf("expect IsMismatchError to return true; got false")
	}

	ee1 := MissingError{}
	runtest(t, "MissingError.Error - basic", "Missing value", ee1.Error())
	ee2 := MissingError{"IDs are different"}
	runtest(t, "MissingError.Error - with msg", "Missing value - IDs are different", ee2.Error())
	if !IsMissingError(ee2) {
		t.Errorf("expect IsMissingError to return true; got false")
	}

	ef1 := NilError{}
	runtest(t, "NilError.Error - basic", "Nil error", ef1.Error())
	ef2 := NilError{Msg: "Missing ID"}
	runtest(t, "NilError.Error - with msg", "Nil error (Missing ID)", ef2.Error())
	ef3 := NilError{Err: ee1}
	runtest(t, "NilError.Error - with error", "Nil error - Missing value", ef3.Error())
	ef4 := NilError{Err: ee1, Msg: "Missing ID"}
	runtest(t, "NilError.Error - with msg and error", "Nil error (Missing ID) - Missing value", ef4.Error())
	if !IsNilError(ef4) {
		t.Errorf("expect IsNilError to return true; got false")
	}

	eg1 := NotFoundError{}
	runtest(t, "NotFoundError.Error - basic", "Entity not found", eg1.Error())
	eg2 := NotFoundError{Kind: "Group"}
	runtest(t, "NotFoundError.Error - with kind", "'Group' entity not found", eg2.Error())
	eg3 := NotFoundError{Err: ee1}
	runtest(t, "NotFoundError.Error - with error", "Entity not found - Missing value", eg3.Error())
	eg4 := NotFoundError{Err: ee1, Kind: "Group"}
	runtest(t, "NotFoundError.Error - with msg and error", "'Group' entity not found - Missing value", eg4.Error())
	if !IsNotFoundError(eg4) {
		t.Errorf("expect IsNotFoundError to return true; got false")
	}
}
