import streamlit as st
import pandas as pd
import plotly.express as px
import os
import psycopg2
from psycopg2.extras import RealDictCursor
import traceback

# Page config
st.set_page_config(
    page_title="名古屋房产信息",
    page_icon="🏠",
    layout="wide"
)

# Database connection with retry
def get_connection():
    max_retries = 3
    for attempt in range(max_retries):
        try:
            if 'DB_HOST' in os.environ:
                conn = psycopg2.connect(
                    host=os.environ.get('DB_HOST'),
                    port=int(os.environ.get('DB_PORT', 5432)),
                    user=os.environ.get('DB_USER'),
                    password=os.environ.get('DB_PASSWORD'),
                    dbname=os.environ.get('DB_NAME'),
                    connect_timeout=10,
                    sslmode='require'
                )
            else:
                conn = psycopg2.connect(
                    host="localhost",
                    port=5432,
                    user="lw",
                    password="",
                    dbname="nagoya_properties",
                    connect_timeout=10
                )
            return conn
        except Exception as e:
            if attempt < max_retries - 1:
                continue
            raise e

# Load data with error handling
@st.cache_data(ttl=60)
def load_data():
    try:
        conn = get_connection()
        cur = conn.cursor(cursor_factory=RealDictCursor)
        cur.execute("""
            SELECT id, source, property_id, listing_type, title, price,
                   price_display, address, area, floor, layout, building_type,
                   construction_year, station_name, walking_minutes, detail_url
            FROM properties ORDER BY id DESC
        """)
        data = cur.fetchall()
        cur.close()
        conn.close()
        return pd.DataFrame(data), None
    except Exception as e:
        return pd.DataFrame(), str(e)

def get_stats():
    try:
        conn = get_connection()
        cur = conn.cursor(cursor_factory=RealDictCursor)
        cur.execute("""
            SELECT source, COUNT(*) as count,
                   COUNT(CASE WHEN title != '' AND address != '' AND area > 0 THEN 1 END) as complete_count,
                   AVG(CASE WHEN area > 0 THEN area END) as avg_area,
                   AVG(CASE WHEN price > 0 THEN price END) as avg_price
            FROM properties GROUP BY source
        """)
        stats = cur.fetchall()
        cur.close()
        conn.close()
        return pd.DataFrame(stats)
    except:
        return pd.DataFrame()

def clean_display(val, max_len=50):
    if pd.notna(val) and val != "" and val != None:
        s = str(val).strip()
        if len(s) > max_len:
            return s[:max_len] + "..."
        return s
    return "N/A"

# Main app
def main():
    st.title("🏠 名古屋房产信息爬虫系统")

    # Load data with error handling
    with st.spinner("加载数据..."):
        df, error = load_data()
        stats = get_stats()

    # Show database connection status
    if error:
        st.error(f"⚠️ 数据库连接失败")
        st.info(f"""
        **错误信息**: {error}

        **请检查Streamlit Cloud的Secrets配置:**

        在 https://share.streamlit.io/ 的你的应用设置中，添加以下Secrets:

        ```toml
        DB_HOST="ep-floral-cherry-a1xz7gdk.ap-southeast-1.aws.neon.tech"
        DB_PORT="5432"
        DB_USER="neondb_owner"
        DB_PASSWORD="npg_UBEigRoV6Dk5"
        DB_NAME="neondb"
        ```

        配置完成后，点击页面右上角的 "Rerun" 或 "Redeploy" 按钮。
        """)
        return

    if df.empty:
        st.warning("📭 数据库中没有数据")
        return

    # Database status
    st.success(f"✅ 已连接数据库 | 共 {len(df)} 条房源")

    # Sidebar filters
    st.sidebar.header("筛选条件")

    all_sources = ['全部'] + sorted(list(df['source'].unique()))
    source_filter = st.sidebar.selectbox("数据源", all_sources)

    all_types = ['全部'] + sorted(list(df['listing_type'].dropna().unique()))
    type_filter = st.sidebar.selectbox("房源类型", all_types)

    # Area filter
    areas = df[df['area'] > 0]['area']
    if len(areas) > 0:
        min_area = float(areas.min())
        max_area = float(areas.max())
        area_range = st.sidebar.slider("面积范围 (㎡)", min_area, max_area, (min_area, max_area))
    else:
        area_range = (0, 200)

    # Price filter
    prices = df[df['price'] > 0]['price']
    if len(prices) > 0:
        min_price = 0
        max_price = float(prices.max()) / 10000
        price_range = st.sidebar.slider("价格范围 (万円)", min_price, max_price, (min_price, max_price))
    else:
        price_range = (0, 100)

    # Apply filters
    filtered_df = df.copy()

    if source_filter != '全部':
        filtered_df = filtered_df[filtered_df['source'] == source_filter]

    if type_filter != '全部':
        filtered_df = filtered_df[filtered_df['listing_type'] == type_filter]

    filtered_df = filtered_df[
        (filtered_df['area'] >= area_range[0]) &
        (filtered_df['area'] <= area_range[1]) &
        (filtered_df['price'] >= price_range[0] * 10000) &
        (filtered_df['price'] <= price_range[1] * 10000)
    ]

    # Stats cards
    st.subheader("数据统计")
    col1, col2, col3, col4 = st.columns(4)

    with col1:
        st.metric("总房源数", len(filtered_df))
    with col2:
        complete_count = len(filtered_df[(filtered_df['title'] != '') & (filtered_df['address'] != '') & (filtered_df['area'] > 0)])
        st.metric("完整数据", complete_count)
    with col3:
        avg_area = filtered_df[filtered_df['area'] > 0]['area'].mean()
        st.metric("平均面积", f"{avg_area:.1f}㎡" if pd.notna(avg_area) and avg_area > 0 else "N/A")
    with col4:
        avg_price = filtered_df[filtered_df['price'] > 0]['price'].mean() / 10000
        st.metric("平均价格", f"{avg_price:.0f}万円" if pd.notna(avg_price) and avg_price > 0 else "N/A")

    # Charts
    if len(filtered_df) > 0:
        col1, col2 = st.columns(2)

        with col1:
            st.subheader("按数据源分布")
            source_counts = filtered_df['source'].value_counts()
            fig_source = px.pie(values=source_counts.values, names=source_counts.index, title="数据源分布")
            st.plotly_chart(fig_source, use_container_width=True)

        with col2:
            st.subheader("面积分布")
            area_data = filtered_df[filtered_df['area'] > 0]['area']
            if len(area_data) > 0:
                fig_area = px.histogram(area_data, bins=20, title="面积分布 (㎡)")
                st.plotly_chart(fig_area, use_container_width=True)

    # Property list
    st.subheader(f"房源列表 ({len(filtered_df)} 条)")

    # Display options
    display_option = st.radio("显示方式", ["表格", "卡片"], horizontal=True)

    if display_option == "表格":
        # Table view
        display_df = filtered_df[[
            'source', 'title', 'address', 'area', 'price_display',
            'layout', 'floor', 'station_name', 'walking_minutes'
        ]].copy()
        display_df.columns = ['数据源', '标题', '地址', '面积(㎡)', '价格', '户型', '楼层', '车站', '步行']
        display_df = display_df.fillna('N/A')
        st.dataframe(display_df, use_container_width=True, height=400)

    else:
        # Card view - simple
        items_per_page = 10
        page = st.number_input("页码", min_value=1, max_value=(len(filtered_df) // items_per_page) + 1, value=1)

        start_idx = (page - 1) * items_per_page
        end_idx = min(start_idx + items_per_page, len(filtered_df))
        page_df = filtered_df.iloc[start_idx:end_idx]

        for _, row in page_df.iterrows():
            with st.container():
                col1, col2, col3 = st.columns([3, 1, 1])

                with col1:
                    st.write(f"**{clean_display(row['title'], 50)}**")
                    st.caption(f"📍 {clean_display(row['address'], 40)}")
                    cols = st.columns(4)
                    cols[0].write(f"🏢 {row['source']}")
                    cols[1].write(f"📏 {row['area']:.1f}㎡" if pd.notna(row['area']) else "📏 N/A")
                    cols[2].write(f"💰 {clean_display(row['price_display'])}")
                    cols[3].write(f"🚪 {clean_display(row['layout'])}")

                with col2:
                    if pd.notna(row['detail_url']) and row['detail_url']:
                        st.markdown(f"<a href='{row['detail_url']}' target='_blank'>详情</a>", unsafe_allow_html=True)

                st.divider()

        # Page navigation
        col1, col2, col3 = st.columns([1, 2, 1])
        with col1:
            if page > 1:
                if st.button("上一页"):
                    st.rerun()
        with col3:
            if end_idx < len(filtered_df):
                if st.button("下一页"):
                    st.rerun()

    # Refresh button
    if st.button("🔄 刷新数据"):
        st.cache_data.clear()
        st.rerun()

if __name__ == "__main__":
    main()
