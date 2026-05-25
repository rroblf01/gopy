import pickle


def main() -> None:
    raw: str = pickle.dumps(["a", "b", "c"])
    obj = pickle.loads(raw)
    print(obj)


if __name__ == "__main__":
    main()
