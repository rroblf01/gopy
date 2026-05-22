import traceback


def main() -> None:
    s = traceback.format_stack()
    print(len(s) >= 0)
    es = traceback.extract_stack()
    print(len(es) >= 0)


if __name__ == "__main__":
    main()
