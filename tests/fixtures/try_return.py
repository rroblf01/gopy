def safe_div(a: int, b: int) -> int:
    try:
        if b == 0:
            raise ValueError("zero")
        return a // b
    except ValueError:
        return -1
    finally:
        print("done div")


def find_first(xs: list[int], target: int) -> int:
    try:
        for x in xs:
            if x == target:
                return x
    finally:
        print("scanned")
    return -1


def void_in_try(n: int) -> None:
    try:
        if n < 0:
            return
        print("positive:", n)
    finally:
        print("clean")


def main() -> None:
    print(safe_div(10, 2))
    print(safe_div(10, 0))
    print(find_first([3, 5, 7], 5))
    print(find_first([1, 2, 3], 9))
    void_in_try(-1)
    void_in_try(4)


if __name__ == "__main__":
    main()
