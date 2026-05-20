import json


def main() -> None:
    with open("/tmp/gopy_json_io.json", "w") as fh:
        fh.write(json.dumps({"a": 1, "b": 2}))
    with open("/tmp/gopy_json_io.json", "r") as fh:
        data = json.load(fh)
        print(type(data).__name__)
    with open("/tmp/gopy_json_io.json", "r") as fh:
        text = fh.read()
        print(len(text) > 0)


if __name__ == "__main__":
    main()
