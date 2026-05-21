import fnmatch


def main() -> None:
    print(fnmatch.fnmatch("foo.py", "*.py"))
    print(fnmatch.fnmatch("bar.txt", "*.py"))
    print(fnmatch.fnmatch("notes.md", "notes.*"))
    names: list[str] = ["a.py", "b.txt", "c.py", "d.md"]
    matched = fnmatch.filter(names, "*.py")
    print(len(matched))
    for n in matched:
        print(n)


if __name__ == "__main__":
    main()
