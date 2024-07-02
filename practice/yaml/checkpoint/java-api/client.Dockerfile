# Use the official OpenJDK image to build the application
FROM openjdk:11-jdk-slim AS build

# Set the working directory
WORKDIR /app

# Copy the Java source code into the container
COPY Client.java .

# Compile the Java source code
RUN javac Client.java

# Package the application into a JAR file
RUN echo "Main-Class: com.example.client.ClientApplication" > manifest.txt && \
    jar cfm client.jar manifest.txt com/example/client/*.class

# Use the official OpenJDK image to run the application
FROM openjdk:11-jre-slim

# Set the working directory
WORKDIR /app

# Copy the JAR file from the build stage
COPY --from=build /app/client.jar .

# Run the JAR file
ENTRYPOINT ["java", "-jar", "client.jar"]
