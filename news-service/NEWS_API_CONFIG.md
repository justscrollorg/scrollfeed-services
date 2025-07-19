# News API Configuration Guide

## Supported News APIs

### 1. GNews (Current Default)
```bash
NEWS_API_BASE_URL=https://gnews.io/api/v4/top-headlines
NEWS_API_KEY=your_gnews_api_key
```
- **Free Tier**: 100 requests/day
- **Pros**: Simple, reliable
- **Cons**: Limited free tier

### 2. NewsAPI.org (Recommended Alternative)
```bash
NEWS_API_BASE_URL=https://newsapi.org/v2/top-headlines
NEWS_API_KEY=your_newsapi_key
```
- **Free Tier**: 1000 requests/day
- **Pros**: More generous free tier, well documented
- **Cons**: None significant

### 3. News Data API
```bash
NEWS_API_BASE_URL=https://newsdata.io/api/1/news
NEWS_API_KEY=your_newsdata_key
```
- **Free Tier**: 200 requests/day
- **Pros**: Good coverage, recent news
- **Cons**: Smaller free tier than NewsAPI

### 4. Currents API
```bash
NEWS_API_BASE_URL=https://api.currentsapi.services/v1/latest-news
NEWS_API_KEY=your_currents_key
```
- **Free Tier**: 600 requests/day
- **Pros**: Good free tier, real-time news
- **Cons**: Less established

## Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| `NEWS_API_KEY` | - | API key (required) |
| `NEWS_API_BASE_URL` | GNews URL | Base URL for the API |
| `NEWS_REGIONS` | us,in,de | Comma-separated country codes |
| `NEWS_MAX_PAGES` | 4 | Pages to fetch per region |
| `NEWS_MAX_ARTICLES` | 33 | Max articles to store per region |
| `NEWS_RATE_LIMIT_SECONDS` | 1 | Delay between API calls |
| `NEWS_FETCH_INTERVAL_HOURS` | 8 | How often to fetch news |

## Migration Steps

1. **Choose your preferred API** from the list above
2. **Sign up and get an API key**
3. **Update environment variables** in your deployment
4. **Test the configuration** locally first
5. **Deploy the updated service**

## Example: Switching to NewsAPI.org

1. Sign up at https://newsapi.org/
2. Get your free API key
3. Update your `.env` or deployment config:
```bash
NEWS_API_KEY=your_newsapi_org_key
NEWS_API_BASE_URL=https://newsapi.org/v2/top-headlines
```
4. Rebuild and deploy the service

## Benefits of This Approach

- **Flexibility**: Easy to switch between providers
- **Configuration**: All settings in environment variables
- **Reliability**: Rate limiting and error handling
- **Scalability**: Configurable fetch intervals and limits
