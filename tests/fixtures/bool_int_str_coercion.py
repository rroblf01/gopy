def main() -> None:
    print(int(True))
    print(int(False))
    a = True
    b = False
    print(int(a) + int(b))
    print(int(a) * 5)
    print(str(True))
    print(str(False))
    print(bool(0))
    print(bool(1))
    print(bool(""))
    print(bool("x"))
    print(bool([]))
    print(bool([1]))
    print(bool({}))
    print(bool({"a": 1}))
    print(bool(None))


if __name__ == "__main__":
    main()
