use std::fmt;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ValidateError {
    Io(String),
    Json(String),
    InvalidRoot,
    UnknownFixtureKind,
    InvalidFixture { kind: String, errors: Vec<String> },
    Cli(String),
}

impl fmt::Display for ValidateError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ValidateError::Io(message) => write!(f, "io error: {message}"),
            ValidateError::Json(message) => write!(f, "json error: {message}"),
            ValidateError::InvalidRoot => write!(f, "fixture root must be a JSON object"),
            ValidateError::UnknownFixtureKind => write!(f, "unknown FORGE-K fixture kind"),
            ValidateError::InvalidFixture { kind, errors } => {
                write!(f, "invalid {kind}: {}", errors.join("; "))
            }
            ValidateError::Cli(message) => write!(f, "{message}"),
        }
    }
}

impl std::error::Error for ValidateError {}

impl From<std::io::Error> for ValidateError {
    fn from(value: std::io::Error) -> Self {
        ValidateError::Io(value.to_string())
    }
}

impl From<serde_json::Error> for ValidateError {
    fn from(value: serde_json::Error) -> Self {
        ValidateError::Json(value.to_string())
    }
}
