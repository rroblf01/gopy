import textwrap


def main() -> None:
    s = "    hello\n    world\n      indented"
    print(textwrap.dedent(s))
    print("---")
    body = "line 1\nline 2\n\nline 4"
    print(textwrap.indent(body, ">>> "))
    print("---")
    print(textwrap.fill("hello world foo bar baz qux quux", 10))


if __name__ == "__main__":
    main()
