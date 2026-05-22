import ipaddress


def main() -> None:
    print(ipaddress.ip_address("192.168.1.1"))
    print(ipaddress.ip_address("::1"))
    print(ipaddress.ip_network("10.0.0.0/24"))


if __name__ == "__main__":
    main()
