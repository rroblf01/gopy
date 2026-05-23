def safe_divide(a: int, b: int) -> int:
    result: int = 0
    try:
        if b == 0:
            raise ZeroDivisionError("divide by zero")
        result = a // b
    except ZeroDivisionError as e:
        print("caught:", e)
        result = 0
    return result


def main() -> None:
    print(safe_divide(10, 2))
    print(safe_divide(7, 0))
    print(safe_divide(20, 4))


if __name__ == "__main__":
    main()
