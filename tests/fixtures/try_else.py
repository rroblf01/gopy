def safe(n: int) -> str:
    result = "init"
    try:
        if n < 0:
            raise ValueError("neg")
    except ValueError:
        result = "caught"
    else:
        result = "ok"
    return result


def main() -> None:
    print(safe(5))
    print(safe(-1))


if __name__ == "__main__":
    main()
