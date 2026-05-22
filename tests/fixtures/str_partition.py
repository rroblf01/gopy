def main() -> None:
    s = "user@example.com"
    head, sep, tail = s.partition("@")
    print(head)
    print(sep)
    print(tail)
    s2 = "no separator here"
    h2, s2v, t2 = s2.partition("@")
    print(h2)
    print(repr(s2v))
    print(repr(t2))


if __name__ == "__main__":
    main()
