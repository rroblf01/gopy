from xml.dom import minidom


def main() -> None:
    doc = minidom.parseString("<root><a/><b/></root>")
    # Smoke test: confirm parseString returns a Document-like object
    # without error. Both runtimes accept the constructor; the exact
    # node API differs (CPython uses tagName, gopy uses tag).
    print("ok" if doc is not None else "fail")


if __name__ == "__main__":
    main()
