def run(which: int) -> str:
    result = "ok"
    try:
        if which == 1:
            raise ValueError("v")
        elif which == 2:
            raise KeyError("k")
        elif which == 3:
            raise RuntimeError("r")
    except (KeyError, ValueError):
        result = "key-or-value"
    except RuntimeError:
        result = "runtime"
    return result


def main() -> None:
    print(run(1))
    print(run(2))
    print(run(3))
    print(run(0))


if __name__ == "__main__":
    main()
