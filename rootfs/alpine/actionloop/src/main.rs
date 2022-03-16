use actix_web::{get, App, HttpResponse, HttpServer, Responder};
use std::process::Command;
use log::warn;
use log::info;



#[get("/")]
async fn hello() -> impl Responder {
    HttpResponse::Ok().body("Hello world!")
}

#[get("/dmesg")]
async fn dmesg() -> impl Responder {
    match Command::new("dmesg").output() {
        Ok(output) => {
            HttpResponse::Ok()
            .content_type("plain/text")
            .body(output.stdout)
        }
        Err(err) => {
            warn!("Failed to run dmesg: {}", err);
            HttpResponse::InternalServerError().finish()
        }
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init();
    info!("Starting web server!!");
    HttpServer::new(|| {
        App::new()
            .service(hello)
            .service(dmesg)
    })
    .bind("0.0.0.0:5000")?
    .run()
    .await
}
