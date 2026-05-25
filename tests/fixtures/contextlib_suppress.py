from contextlib import suppress


def main() -> None:
    seen: list[str] = []

    # IndexError — both CPython and gopy panic on list[100].
    with suppress(IndexError):
        xs: list[int] = [1, 2, 3]
        seen.append("before")
        seen.append(str(xs[100]))
        seen.append("unreachable")
    print("after-index", seen)

    # ValueError raised explicitly — handled by suppress.
    seen2: list[str] = []
    with suppress(ValueError, KeyError):
        seen2.append("before")
        raise ValueError("oops")
    print("after-value", seen2)

    # Different exception class — propagates to outer try.
    try:
        with suppress(KeyError):
            raise ValueError("propagate")
    except ValueError as e:
        print("caught:", str(e))


if __name__ == "__main__":
    main()
