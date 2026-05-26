from string import Formatter


def main() -> None:
    f = Formatter()
    print(f.format("hello {0} and {1}", "alice", "bob"))
    print(f.format("auto {} then {}", "x", "y"))
    print(f.format("escape {{lit}} and {0}", "v"))


if __name__ == "__main__":
    main()
