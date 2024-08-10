FROM python:3.10.0-slim-buster
WORKDIR /app
COPY ray_requirements.txt ray_requirements.txt
COPY ray_checkpoint.py ray_checkpoint.py
RUN pip install --upgrade pip
RUN pip install -r ray_requirements.txt
CMD [  "python", "ray_checkpoint.py" ]