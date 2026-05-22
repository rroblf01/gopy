def call(name: str, **kwargs) -> None:
    print(name)
    for k in sorted(kwargs.keys()):
        print(k, "=", kwargs[k])


def main() -> None:
    call("greet", a=1, b=2, c=3)
    call("solo")


if __name__ == "__main__":
    main()
