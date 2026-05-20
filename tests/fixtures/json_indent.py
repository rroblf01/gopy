import json


def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2}
    print(json.dumps(d, indent=2))
    print("---")
    xs: list[int] = [1, 2, 3]
    print(json.dumps(xs, indent=4))
    print("---")
    print(json.dumps(d))


if __name__ == "__main__":
    main()
