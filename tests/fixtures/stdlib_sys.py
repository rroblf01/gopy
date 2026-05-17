import sys


def main() -> None:
    # sys.argv[0] is the program path in both Python and Go; skip it.
    args: list[str] = sys.argv
    print(len(args) > 0)


if __name__ == "__main__":
    main()
