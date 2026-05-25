from string import Template


def main() -> None:
    t = Template("Hello $name, you owe $amount tokens. ($name)")
    for k in t.get_identifiers():
        print(k)
    print(t.is_valid())

    bad = Template("Hello $")
    print(bad.is_valid())


if __name__ == "__main__":
    main()
