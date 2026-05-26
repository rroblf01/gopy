import weakref


def main() -> None:
    d = weakref.WeakValueDictionary()
    print(len(d))


if __name__ == "__main__":
    main()
