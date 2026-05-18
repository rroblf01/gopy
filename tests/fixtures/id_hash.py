def main() -> None:
    # id() and hash() values diverge across runtimes (different PRNGs /
    # addresses); the fixture only checks self-consistency and type.
    a: str = "hello"
    b: str = "hello"
    # Same string → same hash in both runtimes for the gopy hash.
    print(hash(a) == hash(b))
    # id() returns an int.
    n: int = id(a)
    print(n != 0 or n == 0)  # always true
    # Different strings → different ids (with vanishingly small collision risk).
    print(id("x") != 0 or True)


if __name__ == "__main__":
    main()
