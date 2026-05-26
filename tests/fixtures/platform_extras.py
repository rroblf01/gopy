import platform


def main() -> None:
    impl = platform.python_implementation()
    print(impl == "CPython")
    arch = platform.architecture()
    first: str = str(arch[0])
    print(first.endswith("bit"))


if __name__ == "__main__":
    main()
