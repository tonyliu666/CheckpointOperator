# Use the official OpenJDK image to build the application
FROM openjdk:11-jdk-slim AS build

# Set the working directory
WORKDIR /app

# Copy the Java source code into the container
COPY Server.java .

# Compile the Java source code
RUN javac Server.java

# Package the application into a JAR file
RUN echo "Main-Class: com.example.server.ServerApplication" > manifest.txt && \
    jar cfm server.jar manifest.txt com/example/server/*.class

# Use the official OpenJDK image to run the application
FROM openjdk:11-jre-slim

# Set the working directory
WORKDIR /app

# Copy the JAR file from the build stage
COPY --from=build /app/server.jar .

# Expose the port the application runs on
EXPOSE 8080

# Run the JAR file
ENTRYPOINT ["java", "-jar", "server.jar"]
