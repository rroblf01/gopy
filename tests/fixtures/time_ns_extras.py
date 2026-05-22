import time


def main() -> None:
    print(time.perf_counter_ns() > 0)
    print(time.monotonic_ns() > 0)
    print(time.process_time() >= 0)
    print(time.process_time_ns() > 0)
    print(time.thread_time() >= 0)
    print(time.CLOCK_REALTIME == 0)


if __name__ == "__main__":
    main()
