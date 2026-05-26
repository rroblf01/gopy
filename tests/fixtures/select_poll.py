import select


def main() -> None:
    p = select.poll()
    p.register(0)
    p.register(1)
    p.unregister(0)
    res = p.poll(0)
    print(len(res))


if __name__ == "__main__":
    main()
