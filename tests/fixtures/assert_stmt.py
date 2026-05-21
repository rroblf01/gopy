def safe_divide(a: int, b: int) -> int:
    assert b != 0, "denominator must be nonzero"
    return a // b


def main() -> None:
    assert 1 + 1 == 2
    assert [1, 2, 3]
    print(safe_divide(10, 2))
    try:
        safe_divide(10, 0)
    except Exception as e:
        print(f"caught: {e}")
    try:
        assert False, "bad state"
    except Exception as e:
        print(f"caught: {e}")
    try:
        assert ""
    except Exception as e:
        print("caught empty")


if __name__ == "__main__":
    main()
