import json


def main() -> None:
    s: str = json.dumps([1, 2, 3])
    print(s)
    s2: str = json.dumps({"a": 1, "b": 2})
    # dict iteration order isn't guaranteed in either runtime in general;
    # CPython 3.7+ preserves insertion order, and Go's encoding/json
    # serializes maps with sorted keys. Use a single-key dict to avoid
    # ordering differences.
    s3: str = json.dumps({"x": 42})
    print(s3)
    print(len(s) > 0)
    print(len(s2) > 0)


if __name__ == "__main__":
    main()
