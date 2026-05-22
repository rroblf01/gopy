class CustomError(Exception):
    def __init__(self, code: int, msg: str) -> None:
        super().__init__(msg)
        self.code = code


def faulty(n: int) -> int:
    if n < 0:
        raise CustomError(42, "negative not allowed")
    if n == 0:
        raise ValueError("zero")
    return n * 2


def main() -> None:
    try:
        print(faulty(5))
    except CustomError as e:
        print("custom:", e.code)
    try:
        faulty(-1)
    except CustomError as e:
        print("caught:", e.code)
    try:
        faulty(0)
    except ValueError as e:
        print("value error")
    except Exception as e:
        print("other")


if __name__ == "__main__":
    main()
