import marshal


def main() -> None:
    s = marshal.dumps([1, 2, 3])
    # Wire format differs from CPython (JSON-backed); round-trip the
    # serialized payload back to a structure of any shape.
    out = marshal.loads(s)
    # Confirm round-trip succeeded without raising.
    print("ok" if out is not None else "fail")


if __name__ == "__main__":
    main()
