import streamlit as st

st.set_page_config(page_title="测试", layout="wide")

st.title("🏠 名古屋房产信息 - 测试页面")

st.success("✅ 应用正常运行！")

st.info("""
如果能看到这个页面，说明Streamlit Cloud部署成功。

接下来需要配置数据库连接。
""")

st.write("环境变量检查:")
import os
if 'DB_HOST' in os.environ:
    st.success(f"✅ DB_HOST = {os.environ.get('DB_HOST')}")
    st.success(f"✅ DB_USER = {os.environ.get('DB_USER')}")
    st.success(f"✅ DB_NAME = {os.environ.get('DB_NAME')}")
else:
    st.error("❌ 未检测到数据库配置")
    st.warning("""
    请在Streamlit Cloud的设置中添加以下Secrets:

    ```
    DB_HOST = "ep-floral-cherry-a1xz7gdk.ap-southeast-1.aws.neon.tech"
    DB_PORT = "5432"
    DB_USER = "neondb_owner"
    DB_PASSWORD = "npg_UBEigRoV6Dk5"
    DB_NAME = "neondb"
    ```
    """)

st.write("---")
st.write("依赖库检查:")
try:
    import pandas
    st.success("✅ pandas 可用")
except:
    st.error("❌ pandas 不可用")

try:
    import plotly
    st.success("✅ plotly 可用")
except:
    st.error("❌ plotly 不可用")

try:
    import psycopg2
    st.success("✅ psycopg2 可用")
except:
    st.error("❌ psycopg2 不可用")
