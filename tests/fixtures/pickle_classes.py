import pickle
import os


def main() -> None:
    path = "/tmp/gopy_pickle_classes.pkl"

    with open(path, "wb") as fh:
        p = pickle.Pickler(fh)
        p.dump([1, 2, 3])

    loaded: any = None
    with open(path, "rb") as fh:
        u = pickle.Unpickler(fh)
        loaded = u.load()

    print(loaded is None)

    os.remove(path)


if __name__ == "__main__":
    main()
