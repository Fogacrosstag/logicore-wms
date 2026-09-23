FROM maven:3.9.9-eclipse-temurin-21 AS build
WORKDIR /src
COPY pom.xml .
COPY libs/java-common libs/java-common
COPY services services
RUN mvn -B -pl services/reservation-service -am package -DskipTests
FROM eclipse-temurin:21-jre-alpine
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /src/services/reservation-service/target/reservation-service-1.0.0.jar app.jar
USER app
EXPOSE 8080
ENTRYPOINT ["java","-XX:MaxRAMPercentage=70","-jar","app.jar"]
