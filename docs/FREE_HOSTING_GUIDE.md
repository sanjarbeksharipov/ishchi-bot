# Free Server Hosting Options for Telegram Bot

This guide lists free hosting platforms where you can deploy your Telegram bot with Docker support.

## 🌟 Recommended Options

### 1. **Render** ⭐ (Highly Recommended)
**Website:** https://render.com

**Free Tier:**
- 750 hours/month of free runtime
- PostgreSQL database with 90-day retention
- Automatic deploys from Git
- SSL certificates included
- Good uptime

**Pros:**
- ✅ Native Docker support
- ✅ Free PostgreSQL database
- ✅ Easy to use
- ✅ Automatic deploys from GitHub/GitLab
- ✅ Environment variables management
- ✅ Good documentation

**Cons:**
- ❌ Service sleeps after 15 min of inactivity (can use cron-job.org to keep alive)
- ❌ Limited to 750 hours/month

**Deployment Steps:**
1. Push code to GitHub/GitLab
2. Connect repository to Render
3. Select "Web Service" with Docker
4. Add PostgreSQL database
5. Configure environment variables
6. Deploy

---

### 2. **Railway** ⭐
**Website:** https://railway.app

**Free Tier:**
- $5 credit/month (enough for small bots)
- PostgreSQL included
- No sleep time
- Easy deployment

**Pros:**
- ✅ Excellent Docker support
- ✅ Free PostgreSQL
- ✅ No auto-sleep
- ✅ Very easy to use
- ✅ Good performance

**Cons:**
- ❌ Limited monthly credit ($5/month)
- ❌ May need credit card for verification

**Deployment Steps:**
1. Sign up with GitHub
2. Create new project
3. Deploy from GitHub repo
4. Add PostgreSQL plugin
5. Set environment variables
6. Deploy

---

### 3. **Fly.io**
**Website:** https://fly.io

**Free Tier:**
- 3 shared-cpu-1x VMs
- 3GB persistent storage
- PostgreSQL support
- 160GB bandwidth

**Pros:**
- ✅ Excellent Docker support
- ✅ Good free tier
- ✅ No auto-sleep
- ✅ Global deployment
- ✅ CLI tools

**Cons:**
- ❌ Requires credit card (no charges for free tier)
- ❌ Steeper learning curve

**Deployment Steps:**
```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login
flyctl auth login

# Launch app
flyctl launch

# Create PostgreSQL
flyctl postgres create

# Set secrets
flyctl secrets set BOT_TOKEN=your_token

# Deploy
flyctl deploy
```

---

### 4. **Oracle Cloud Free Tier** ⭐ (Best for 24/7 uptime)
**Website:** https://www.oracle.com/cloud/free/

**Free Tier (Always Free):**
- 2 AMD VMs with 1GB RAM each OR 4 ARM VMs with 24GB RAM total
- 200GB storage
- 10TB bandwidth/month
- No time limits
- True 24/7 uptime

**Pros:**
- ✅ Very generous free tier
- ✅ No auto-sleep
- ✅ True always-free (not trial)
- ✅ Full VM control
- ✅ Can host multiple bots

**Cons:**
- ❌ Requires credit card
- ❌ More complex setup
- ❌ Need to manage your own server

**Deployment Steps:**
1. Create Oracle Cloud account
2. Create VM instance (Ubuntu)
3. Install Docker and Docker Compose
4. Clone your repository
5. Run `docker-compose up -d`

---

### 5. **Heroku** (Limited Free Option)
**Website:** https://www.heroku.com

**Note:** Heroku removed their free tier in November 2022, but students can get credits through GitHub Student Pack.

**Student Tier (with GitHub Student Pack):**
- Free dynos for students
- PostgreSQL support
- Easy deployment

---

### 6. **Google Cloud Run**
**Website:** https://cloud.google.com/run

**Free Tier:**
- 2 million requests/month
- 360,000 GB-seconds/month
- 180,000 vCPU-seconds/month

**Pros:**
- ✅ Good Docker support
- ✅ Generous free tier
- ✅ Scales to zero

**Cons:**
- ❌ Requires credit card
- ❌ Complex for beginners
- ❌ Service sleeps when idle
- ❌ Not ideal for long-polling bots

---

### 7. **AWS Free Tier** (ECS/Fargate)
**Website:** https://aws.amazon.com/free/

**Free Tier (12 months):**
- 750 hours EC2 t2.micro
- RDS PostgreSQL 750 hours
- Complex but powerful

**Pros:**
- ✅ Professional-grade infrastructure
- ✅ Generous free tier

**Cons:**
- ❌ Complex setup
- ❌ Only free for 12 months
- ❌ Requires credit card
- ❌ Can accidentally incur charges

---

### 8. **Koyeb**
**Website:** https://www.koyeb.com

**Free Tier:**
- 2 web services
- 1 instance per service
- Automatic deploys

**Pros:**
- ✅ Easy Docker deployment
- ✅ Auto-deploy from Git
- ✅ Free SSL

**Cons:**
- ❌ Limited resources
- ❌ Auto-sleep after inactivity

---

## 📊 Comparison Table

| Platform | Docker Support | Database | Auto-Sleep | Setup Difficulty | Best For |
|----------|---------------|----------|------------|------------------|----------|
| **Render** | ✅ Excellent | ✅ Free PostgreSQL | ⚠️ Yes (15 min) | ⭐ Easy | Beginners |
| **Railway** | ✅ Excellent | ✅ Free PostgreSQL | ✅ No | ⭐ Easy | Small bots |
| **Fly.io** | ✅ Excellent | ✅ Supported | ✅ No | ⭐⭐ Medium | Production |
| **Oracle Cloud** | ✅ Full control | ⚠️ Self-managed | ✅ No | ⭐⭐⭐ Hard | 24/7 uptime |
| **Google Cloud Run** | ✅ Native | ⚠️ Separate | ⚠️ Yes | ⭐⭐ Medium | Serverless |
| **Koyeb** | ✅ Good | ❌ No | ⚠️ Yes | ⭐ Easy | Testing |

---

## 🚀 Quick Recommendation

**For Beginners:** Start with **Render** or **Railway**
- Easiest to set up
- Good free tier
- Includes database

**For 24/7 Uptime:** Use **Oracle Cloud** or **Fly.io**
- No auto-sleep
- Better performance
- More reliable

**For Multiple Bots:** Use **Oracle Cloud Free Tier**
- Can host multiple applications
- Very generous resources

---

## 🎯 My Top 3 Recommendations

### 🥇 1st Choice: Railway
- Easiest to use
- No sleep time
- Free PostgreSQL
- Perfect for Telegram bots

### 🥈 2nd Choice: Render
- Great free tier
- Easy deployment
- Only downside: auto-sleep (easily solved with cron-job.org)

### 🥉 3rd Choice: Fly.io
- Excellent for production
- No auto-sleep
- Good documentation

---

## 💡 Tips to Maximize Free Tiers

1. **Use multiple platforms:** Deploy different services on different platforms
2. **Keep services alive:** Use cron-job.org or UptimeRobot to ping your service
3. **Optimize resources:** Use small Docker images (Alpine-based)
4. **Monitor usage:** Set up alerts for usage limits
5. **Use webhooks instead of polling:** Reduces resource usage

---

## 🔧 Deployment Steps for Render (Recommended)

1. **Push your code to GitHub/GitLab**
   ```bash
   git init
   git add .
   git commit -m "Initial commit"
   git remote add origin <your-repo-url>
   git push -u origin main
   ```

2. **Sign up at Render.com**
   - Use GitHub/GitLab to sign up

3. **Create PostgreSQL Database**
   - Click "New +" → "PostgreSQL"
   - Name: `ishchi-bot-db`
   - Free plan
   - Click "Create Database"
   - Save the "Internal Database URL"

4. **Create Web Service**
   - Click "New +" → "Web Service"
   - Connect your repository
   - Select your bot repository
   - Name: `ishchi-bot`
   - Runtime: Docker
   - Free plan

5. **Add Environment Variables**
   ```
   BOT_TOKEN=your_bot_token
   DB_HOST=<from database internal URL>
   DB_PORT=5432
   DB_USER=<from database>
   DB_PASSWORD=<from database>
   DB_NAME=<from database>
   APP_ENV=production
   LOG_LEVEL=info
   BOT_ADMIN_IDS=your_admin_id
   ```

6. **Deploy**
   - Click "Create Web Service"
   - Wait for deployment to complete

7. **Keep Service Alive** (Optional)
   - Sign up at cron-job.org
   - Create a job to ping your service every 10 minutes

---

## 📝 Notes

- **Telegram bots** work well with free tiers because they use minimal resources
- **Long-polling** is better for free tiers than webhooks (no need for public URLs)
- **Always set up monitoring** to track uptime and errors
- **Use environment variables** for sensitive data, never commit them to Git

---

## ⚠️ Important Security Notes

1. Never commit `.env` file to Git
2. Use strong database passwords
3. Limit admin access properly
4. Keep your bot token secret
5. Regularly update dependencies

---

## 🆘 Need Help?

If you have issues deploying:
1. Check the platform's documentation
2. Look at deployment logs
3. Verify all environment variables are set
4. Check database connectivity
5. Ensure migrations are running

Good luck with your deployment! 🚀
