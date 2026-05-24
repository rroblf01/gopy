from pathlib import Path


def main() -> None:
    p = Path("foo/bar.txt")
    print(p.is_absolute())
    print(p.with_suffix(".log"))
    print(p.with_name("baz.txt"))
    q = Path("/tmp/x.txt")
    print(q.is_absolute())


if __name__ == "__main__":
    main()
