import streamlit as st
import pandas as pd
import plotly.express as px
import os
import psycopg2
from psycopg2.extras import RealDictCursor

# Page config
st.set_page_config(
    page_title="名古屋房产信息",
    page_icon="🏠",
    layout="wide"
)

# Database connection
def get_connection():
    if 'DB_HOST' in os.environ:
        return psycopg2.connect(
            host=os.environ.get('DB_HOST'),
            port=int(os.environ.get('DB_PORT', 5432)),
            user=os.environ.get('DB_USER'),
            password=os.environ.get('DB_PASSWORD'),
            dbname=os.environ.get('DB_NAME')
        )
    else:
        return psycopg2.connect(
            host="localhost",
            port=5432,
            user="lw",
            password="",
            dbname="nagoya_properties"
        )

# Load data
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
        return pd.DataFrame(data)
    except Exception as e:
        st.error(f"数据库连接失败: {e}")
        return pd.DataFrame()

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
    except Exception as e:
        return pd.DataFrame()

def safe_str(val):
    return str(val) if pd.notna(val) else ""

def clean_display(val, max_len=50):
    if pd.notna(val) and val != "":
        s = str(val)
        if len(s) > max_len:
            return s[:max_len] + "..."
        return s
    return "N/A"

# Main app
def main():
    st.title("🏠 名古屋房产信息爬虫系统")

    with st.spinner("加载数据..."):
        df = load_data()
        stats = get_stats()

    if df.empty:
        st.error("""
        ### 无法连接到数据库

        请在Streamlit Cloud设置中添加以下Secrets:

        ```toml
        DB_HOST="ep-floral-cherry-a1xz7gdk.ap-southeast-1.aws.neon.tech"
        DB_PORT="5432"
        DB_USER="neondb_owner"
        DB_PASSWORD="npg_UBEigRoV6Dk5"
        DB_NAME="neondb"
        ```
        """)
        return

    # Sidebar filters
    st.sidebar.header("筛选条件")

    all_sources = ['全部'] + list(df['source'].unique())
    source_filter = st.sidebar.selectbox("数据源", all_sources)

    all_types = ['全部'] + list(df['listing_type'].dropna().unique())
    type_filter = st.sidebar.selectbox("房源类型", all_types)

    min_area = float(df['area'].min()) if df['area'].min() > 0 else 0
    max_area = float(df['area'].max()) if df['area'].max() > 0 else 200
    area_range = st.sidebar.slider("面积范围 (㎡)", min_area, max_area, (min_area, max_area))

    min_price = 0
    max_price = float(df['price'].max()) if df['price'].max() > 0 else 100
    price_range = st.sidebar.slider("价格范围 (万円)", min_price, max_price, (min_price, max_price))

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
        st.metric("平均面积", f"{avg_area:.1f}㎡" if avg_area > 0 else "N/A")
    with col4:
        avg_price = filtered_df[filtered_df['price'] > 0]['price'].mean() / 10000
        st.metric("平均价格", f"{avg_price:.1f}万円" if avg_price > 0 else "N/A")

    # Charts
    col1, col2 = st.columns(2)

    with col1:
        st.subheader("按数据源分布")
        source_counts = filtered_df['source'].value_counts()
        if len(source_counts) > 0:
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
    display_option = st.radio("显示方式", ["卡片", "表格"], horizontal=True)

    if display_option == "表格":
        # Table view (safer, no custom HTML)
        display_df = filtered_df[[
            'source', 'title', 'address', 'area', 'price_display',
            'layout', 'floor', 'station_name', 'walking_minutes'
        ]].copy()
        display_df.columns = ['数据源', '标题', '地址', '面积(㎡)', '价格', '户型', '楼层', '车站', '步行(分)']
        display_df = display_df.fillna('N/A')
        st.dataframe(display_df, use_container_width=True, height=400)

    else:
        # Card view - using st.container instead of custom HTML
        for _, row in filtered_df.iterrows():
            with st.container():
                col1, col2 = st.columns([3, 1])

                with col1:
                    title = clean_display(row['title'], 60)
                    st.write(f"**{title}**")
                    st.caption(f"📍 {clean_display(row['address'], 40)}")
                    st.write(f"🏢 {row['source']} | {'出租' if row['listing_type'] == 'rent' else '出售'}")
                    st.write(f"📏 {row['area']:.2f}㎡  💰 {clean_display(row['price_display'])}")

                with col2:
                    if pd.notna(row['detail_url']) and row['detail_url']:
                        st.link_button("查看详情", row['detail_url'])

                st.divider()

    # Detail view
    if len(filtered_df) > 0:
        st.subheader("房源详情")
        selected_idx = st.selectbox(
            "选择房源查看详情",
            options=range(len(filtered_df)),
            format_func=lambda i: f"{clean_display(filtered_df.iloc[i]['title'], 40)} - {filtered_df.iloc[i]['source']}"
        )

        if selected_idx is not None:
            prop = filtered_df.iloc[selected_idx]

            col1, col2 = st.columns(2)

            with col1:
                st.write(f"**标题**: {clean_display(prop['title'])}")
                st.write(f"**数据源**: {prop['source']}")
                st.write(f"**房源ID**: {prop['property_id']}")
                st.write(f"**类型**: {'出租' if prop['listing_type'] == 'rent' else '出售'}")
                st.write(f"**地址**: {clean_display(prop['address'])}")

            with col2:
                st.write(f"**价格**: {clean_display(prop['price_display'])}")
                st.write(f"**面积**: {prop['area']:.2f}㎡" if pd.notna(prop['area']) and prop['area'] > 0 else "**面积**: N/A")
                st.write(f"**户型**: {clean_display(prop['layout'])}")
                st.write(f"**楼层**: {clean_display(prop['floor'])}")
                st.write(f"**建筑类型**: {clean_display(prop['building_type'])}")

            st.write(f"**建造年份**: {prop['construction_year']}" if pd.notna(prop['construction_year']) else "**建造年份**: N/A")
            st.write(f"**最近车站**: {clean_display(prop['station_name'])}")
            st.write(f"**步行时间**: {prop['walking_minutes']} 分钟" if pd.notna(prop['walking_minutes']) else "**步行时间**: N/A")

            if pd.notna(prop['detail_url']) and prop['detail_url']:
                st.link_button("查看原始链接", prop['detail_url'])

    # Refresh button
    if st.button("🔄 刷新数据"):
        st.cache_data.clear()
        st.rerun()

if __name__ == "__main__":
    main()
