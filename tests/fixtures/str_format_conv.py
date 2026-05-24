def main() -> None:
    print("{!r}".format("hello"))
    print("{!s}".format(42))
    print("{0!r} and {1!s}".format("x", 42))
    print("{name!r}".format(name="alice"))
    print("{0!r:>15}".format("padded"))


if __name__ == "__main__":
    main()
