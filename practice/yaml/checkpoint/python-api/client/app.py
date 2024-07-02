import requests
import time

def main():
    response = requests.get('http://flask-server:5000/hello')
    print(response.text)

if __name__ == '__main__':
    # every seconds access the main function
    while True:
        main()
        time.sleep(1)
        
