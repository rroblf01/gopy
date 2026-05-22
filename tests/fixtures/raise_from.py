def do_work() -> None:
    try:
        raise Exception("inner")
    except Exception as v:
        raise Exception("wrapped from inner") from v


def main() -> None:
    try:
        do_work()
    except Exception as e:
        print(e)


if __name__ == "__main__":
    main()
