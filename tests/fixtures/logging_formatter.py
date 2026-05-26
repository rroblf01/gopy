import logging


def main() -> None:
    f = logging.Formatter("%(name)s :: %(message)s")
    print("ok" if f is not None else "fail")


if __name__ == "__main__":
    main()
