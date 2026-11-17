import { useInfiniteQuery, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiBaseUrl, apiFetch } from './api';
import type { CommentListResponse, CourseDetailCourse as Course, CourseDetailResponse, PostListResponse, PublicPost, PublicReview, ReviewListResponse, TopicTagListResponse } from './src/api/types';
import { useState } from 'react';
import { confirmDiscard, useUnsavedChanges, type LeaveState } from './src/ui/unsavedChanges';
import { ActivityIndicator, Alert, Image, ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';

const green='#315f4a';const ink='#202722';
type Detail=CourseDetailResponse;
type Review=PublicReview;
type Post=PublicPost;
type CommunityPolicy={content_review_sla_hours:number|null;community_guidelines:string};
function useCommunityPolicy(token:string){return useQuery({queryKey:['community-guidelines'],queryFn:async()=>{const r=await apiFetch('/api/v1/community-guidelines',token);if(!r.ok)throw new Error('社区信息暂不可用');return await r.json() as CommunityPolicy;}})}
function CourseCompletionButton({token,course,canEdit}:{token:string;course:Course;canEdit:boolean}){const qc=useQueryClient();const courses=useQuery({queryKey:['my-courses'],queryFn:async()=>{const r=await apiFetch('/api/v1/users/me/courses',token);if(!r.ok)throw new Error();return(await r.json() as {items:{id:number}[]}).items;}});const complete=!!courses.data?.some(x=>x.id===course.id);const toggle=async()=>{const r=await apiFetch(`/api/v1/courses/${course.id}/completion`,token,{method:complete?'DELETE':'POST'});if(!r.ok){Alert.alert('无法更新','请稍后重试');return}await qc.invalidateQueries({queryKey:['my-courses']})};if(!canEdit)return null;return <TouchableOpacity onPress={()=>void toggle()}><Text style={s.report}>{complete?'✓ 已修读课程 · 移除':'＋ 标记为已修读'}</Text></TouchableOpacity>}

type DetailActions = {
 onBack: () => void;
 onCompose: (course: Course) => void;
 onDirtyChange: (dirty: LeaveState) => void;
};

export function CourseDetail({courseId,token,onBack,onCompose,onDirtyChange}:DetailActions&{courseId:number;token:string}){
 const detail = useQuery({
  queryKey: ['course', courseId],
  queryFn: async () => {
   const response = await apiFetch(`/api/v1/courses/${courseId}`, token);
   if (!response.ok) throw new Error('课程信息暂不可用');
   return (await response.json()) as Detail;
  },
 });
 if (detail.data?.course) {
  return <CourseDetailContent key={courseId} detail={detail.data} token={token} onBack={onBack} onCompose={onCompose} onDirtyChange={onDirtyChange}/>;
 }
 return <View style={s.screen}>
  <View style={s.header}><TouchableOpacity onPress={onBack}><Text style={s.back}>‹ 返回</Text></TouchableOpacity><Text style={s.headerTitle}>课程详情</Text></View>
  {detail.isPending ? <ActivityIndicator color={green}/> : (
   <TouchableOpacity style={s.content} onPress={() => void detail.refetch()}>
    <Text style={s.hint}>课程信息暂不可用，点击重试</Text>
   </TouchableOpacity>
  )}
 </View>;
}

function CourseDetailContent({detail,token,onBack,onCompose,onDirtyChange}:DetailActions&{detail:Detail;token:string}){
 const { course, ratings, review_count } = detail;
 const [tab,setTab]=useState<'评价'|'经验'>('评价');const [reviewOpen,setReviewOpen]=useState(false);const [reviewDirty,setReviewDirty]=useState<LeaveState>(false);const [reviewTag,setReviewTag]=useState('');const [postTag,setPostTag]=useState('');const [postYear,setPostYear]=useState('');const [postSort,setPostSort]=useState<'newest'|'popular'>('newest');const [notice,setNotice]=useState('');const [correctionOpen,setCorrectionOpen]=useState(false);const [courseName,setCourseName]=useState(course.course_name);const [professor,setProfessor]=useState(course.professor);const [department,setDepartment]=useState(course.department);const [semester,setSemester]=useState(course.semester);const [credits,setCredits]=useState(String(course.credits));const [correctionMessage,setCorrectionMessage]=useState('');const qc=useQueryClient();const availableTags=useTags(token);
 useUnsavedChanges(reviewDirty, onDirtyChange);
 const currentUser=useQuery({queryKey:['current-user'],queryFn:async()=>{const r=await apiFetch('/api/v1/auth/me',token);if(!r.ok)throw new Error('账号状态暂不可用');return await r.json() as {status:string}}});const canPublish=currentUser.data?.status==='verified';
 const reviews=useInfiniteQuery({queryKey:['reviews',course.id,reviewTag],initialPageParam:1,queryFn:async({pageParam})=>{const params=new URLSearchParams({page:String(pageParam),page_size:'100'});if(reviewTag)params.set('tag',reviewTag);const r=await apiFetch(`/api/v1/courses/${course.id}/reviews?${params}`,token);if(!r.ok)throw new Error('评价暂不可用');return await r.json() as ReviewListResponse;},getNextPageParam:p=>p.items?.length===(p.page_size??100)?(p.page??1)+1:undefined,enabled:tab==='评价'});
 const posts=useInfiniteQuery({queryKey:['posts',course.id,postYear,postTag,postSort],initialPageParam:1,queryFn:async({pageParam})=>{const params=new URLSearchParams({page:String(pageParam),page_size:'100',sort:postSort});if(postYear)params.set('year',postYear);if(postTag)params.set('tag',postTag);const r=await apiFetch(`/api/v1/courses/${course.id}/posts?${params}`,token);if(!r.ok)throw new Error('经验帖暂不可用');return await r.json() as PostListResponse;},getNextPageParam:p=>p.items?.length===(p.page_size??100)?(p.page??1)+1:undefined,enabled:tab==='经验'});
 const reviewItems=(reviews.data?.pages.flatMap(page=>page.items??[])??[]).filter((item):item is NonNullable<typeof item>=>item!==null);const postItems=(posts.data?.pages.flatMap(page=>page.items??[])??[]).filter((item):item is NonNullable<typeof item>=>item!==null);
 return <View style={s.screen}><View style={s.header}><TouchableOpacity onPress={onBack}><Text style={s.back}>‹ 返回</Text></TouchableOpacity><Text style={s.headerTitle}>课程详情</Text></View><ScrollView contentContainerStyle={s.content} keyboardShouldPersistTaps="handled">
  <Text style={s.code}>{course.course_code} · {course.semester}</Text><Text style={s.title}>{course.course_name}</Text><Text style={s.meta}>{course.professor||'教授待补充'} · {course.department} · {course.credits} 学分</Text><CourseCompletionButton token={token} course={course} canEdit={canPublish}/>{canPublish&&<TouchableOpacity onPress={()=>setCorrectionOpen(v=>!v)}><Text style={s.report}>纠正课程资料</Text></TouchableOpacity>}{correctionOpen&&<View style={s.form}><TextInput value={courseName} onChangeText={setCourseName} placeholder="课程名称" style={s.input}/><TextInput value={professor} onChangeText={setProfessor} placeholder="教授" style={s.input}/><TextInput value={semester} onChangeText={setSemester} placeholder="学期" style={s.input}/><TextInput value={department} onChangeText={setDepartment} placeholder="所属专业" style={s.input}/><TextInput value={credits} onChangeText={setCredits} keyboardType="decimal-pad" placeholder="学分" style={s.input}/><TouchableOpacity style={s.secondary} onPress={()=>void (async()=>{const fields:Record<string,string|number>={};if(courseName!==course.course_name)fields.course_name=courseName;if(professor!==course.professor)fields.professor=professor;if(semester!==course.semester)fields.semester=semester;if(department!==course.department)fields.department=department;const creditValue=credits.trim()===''?null:Number(credits);if(creditValue!==null&&Number.isFinite(creditValue)&&creditValue!==course.credits)fields.credits=creditValue;if(!Object.keys(fields).length){setCorrectionMessage('请先修改一个有效字段');return;}const response=await apiFetch(`/api/v1/courses/${course.id}/corrections`,token,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(fields)});const data=await response.json() as {error?:string};if(response.ok){setCorrectionOpen(false);setCorrectionMessage('建议已提交给运营人员校对。')}else setCorrectionMessage(data.error||'提交失败');})()}><Text style={s.report}>提交纠错建议</Text></TouchableOpacity>{!!correctionMessage&&<Text style={s.privacy}>{correctionMessage}</Text>}</View>}
  <View style={s.ratingCard}><Text style={s.section}>课程评分 · {review_count} 条有效评价</Text><Text style={s.meta}>{review_count ? `给分 ${ratings?.grading.toFixed(1)}　工作量 ${ratings?.workload.toFixed(1)}　考核 ${ratings?.assessment.toFixed(1)}　实用性 ${ratings?.usefulness.toFixed(1)}` : '暂无评分'}</Text></View>
  <View style={s.tabs}>{(['评价','经验'] as const).map(x=><TouchableOpacity key={x} onPress={()=>{if(x!==tab)confirmDiscard(reviewDirty,()=>{setTab(x);setReviewOpen(false);});}} style={[s.tab,tab===x&&s.tabOn]}><Text style={[s.tabText,tab===x&&s.tabTextOn]}>{x}</Text></TouchableOpacity>)}</View>
  {tab==='评价'&&canPublish&&<TouchableOpacity style={s.action} onPress={()=>confirmDiscard(reviewDirty,()=>{setReviewOpen(v=>!v);setNotice('');})}><Text style={s.actionText}>{reviewOpen?'收起评价表单':'写课程评价'}</Text></TouchableOpacity>}
  {tab==='经验'&&canPublish&&<TouchableOpacity style={s.action} onPress={()=>onCompose(course)}><Text style={s.actionText}>分享修课经验</Text></TouchableOpacity>}
  {tab==='评价'&&<><Text style={s.meta}>按评价标签筛选</Text><ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={s.scoreButtons}><TouchableOpacity onPress={()=>setReviewTag('')}><Text style={[s.chip,!reviewTag&&s.chipOn]}>全部</Text></TouchableOpacity>{availableTags.map(t=><TouchableOpacity key={t} onPress={()=>setReviewTag(reviewTag===t?'':t)}><Text style={[s.chip,reviewTag===t&&s.chipOn]}>{t}</Text></TouchableOpacity>)}</ScrollView></>}
  {tab==='经验'&&<><Text style={s.meta}>筛选经验</Text><ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={s.scoreButtons}><TouchableOpacity onPress={()=>setPostYear('')}><Text style={[s.chip,!postYear&&s.chipOn]}>全部年份</Text></TouchableOpacity><TouchableOpacity onPress={()=>setPostYear('unspecified')}><Text style={[s.chip,postYear==='unspecified'&&s.chipOn]}>年份未注明</Text></TouchableOpacity>{Array.from({length:new Date().getFullYear()-1990+1},(_,i)=>new Date().getFullYear()-i).map(y=><TouchableOpacity key={y} onPress={()=>setPostYear(String(y))}><Text style={[s.chip,postYear===String(y)&&s.chipOn]}>{y}</Text></TouchableOpacity>)}</ScrollView><ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={s.scoreButtons}><TouchableOpacity onPress={()=>setPostSort('newest')}><Text style={[s.chip,postSort==='newest'&&s.chipOn]}>最新</Text></TouchableOpacity><TouchableOpacity onPress={()=>setPostSort('popular')}><Text style={[s.chip,postSort==='popular'&&s.chipOn]}>最热门</Text></TouchableOpacity>{availableTags.map(t=><TouchableOpacity key={t} onPress={()=>setPostTag(postTag===t?'':t)}><Text style={[s.chip,postTag===t&&s.chipOn]}>{t}</Text></TouchableOpacity>)}</ScrollView></>}
  {currentUser.data?.status==='alumni_readonly'&&<Text style={s.privacy}>校友只读账号可以浏览课程内容，不能发布或互动。</Text>}
  {!!notice&&<Text style={s.notice}>{notice}</Text>}
  {reviewOpen&&<ReviewForm token={token} course={course} onDirtyChange={setReviewDirty} onDone={async()=>{setReviewOpen(false);setNotice('评价已提交，审核通过后会公开。');await qc.invalidateQueries({queryKey:['course',course.id]});await qc.invalidateQueries({queryKey:['reviews',course.id]});}}/>}
  {tab==='评价'&&(reviews.isLoading?<ActivityIndicator color={green}/>:reviews.error?<TouchableOpacity onPress={()=>void reviews.refetch()}><Text style={s.hint}>评价暂时无法加载，点击重试</Text></TouchableOpacity>:reviewItems.length?reviewItems.map(x=><View key={x.id} style={s.card}><Text style={s.cardTitle}>{x.display_name} · {x.semester}</Text><Text style={s.meta}>给分 {x.grading} · 工作量 {x.workload} · 考核 {x.assessment} · 实用性 {x.usefulness}</Text><Text style={s.body}>{x.body||'未填写文字说明'}</Text><Text style={s.meta}>{x.tags?.join('　')??''}</Text>{canPublish&&<ReportButton token={token} target="review" id={x.id}/>}</View>):<Text style={s.empty}>暂无已发布评价</Text>)}
  {tab==='评价'&&reviews.hasNextPage&&<TouchableOpacity disabled={reviews.isFetchingNextPage} style={s.secondary} onPress={()=>void reviews.fetchNextPage()}><Text style={s.report}>{reviews.isFetchingNextPage?'正在加载…':'加载更多评价'}</Text></TouchableOpacity>}
  {tab==='经验'&&(posts.isLoading?<ActivityIndicator color={green}/>:posts.error?<TouchableOpacity onPress={()=>void posts.refetch()}><Text style={s.hint}>经验帖暂时无法加载，点击重试</Text></TouchableOpacity>:postItems.length?postItems.map(x=><View key={x.id} style={s.card}><Text style={s.cardTitle}>{x.title}</Text><Text style={s.meta}>{x.display_name} · {x.attendance_year_label} · ♥ {x.like_count}</Text><Text style={s.body}>{x.body}</Text>{x.image_urls?.map(uri=><Image key={uri} source={{uri:`${apiBaseUrl}${uri}`,headers:{Authorization:`Bearer ${token}`}}} style={s.image}/>)}<PostInteractions token={token} post={x} canInteract={!!canPublish}/>{canPublish&&<ReportButton token={token} target="post" id={x.id}/>}</View>):<Text style={s.empty}>暂无已发布经验帖</Text>)}
  {tab==='经验'&&posts.hasNextPage&&<TouchableOpacity disabled={posts.isFetchingNextPage} style={s.secondary} onPress={()=>void posts.fetchNextPage()}><Text style={s.report}>{posts.isFetchingNextPage?'正在加载…':'加载更多经验帖'}</Text></TouchableOpacity>}
 </ScrollView></View>;
}

function ReviewForm({token,course,onDone,onDirtyChange}:{token:string;course:Course;onDone:()=>void;onDirtyChange:(dirty:LeaveState)=>void}){
 const [scores,setScores]=useState([3,3,3,3]);const [body,setBody]=useState('');const [busy,setBusy]=useState(false);const [tags,setTags]=useState<string[]>([]);
 useUnsavedChanges(busy ? 'busy' : !!body || tags.length > 0 || scores.some(score=>score!==3),onDirtyChange);
 const available=useTags(token);const policy=useCommunityPolicy(token);const labels=['给分体验','工作量','考核方式','实用性'];const submit=async()=>{setBusy(true);try{const r=await apiFetch(`/api/v1/courses/${course.id}/reviews`,token,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({grading:scores[0],workload:scores[1],assessment:scores[2],usefulness:scores[3],tags,body})});const data=await r.json() as {error?:string};if(!r.ok)throw new Error(data.error||'提交失败');onDone();}catch(e){Alert.alert('无法提交',e instanceof Error?e.message:'请稍后再试');}finally{setBusy(false);}};
 return <View style={s.form}><Text style={s.section}>匿名课程评价</Text>{labels.map((label,i)=><View key={label} style={s.scoreRow}><Text style={s.meta}>{label}　{scores[i]}</Text><View style={s.scoreButtons}>{[1,2,3,4,5].map(n=><TouchableOpacity key={n} onPress={()=>setScores(v=>v.map((x,j)=>j===i?n:x))}><Text style={[s.score,scores[i]===n&&s.scoreOn]}>{n}</Text></TouchableOpacity>)}</View></View>)}<Text style={s.meta}>评价标签（选填）</Text><TagChooser values={available} selected={tags} onChange={setTags}/><TextInput value={body} onChangeText={setBody} multiline maxLength={2000} placeholder="补充你的课程体验（选填）" style={s.input}/><Text style={s.privacy}>评价默认匿名展示，提交后需审核。{policy.data?.content_review_sla_hours?`预计处理时限：${policy.data.content_review_sla_hours} 小时。`:''}</Text>{!!policy.data?.community_guidelines&&<Text style={s.privacy}>{policy.data.community_guidelines}</Text>}<TouchableOpacity disabled={busy} style={s.action} onPress={()=>void submit()}><Text style={s.actionText}>{busy?'提交中…':'提交评价'}</Text></TouchableOpacity></View>;
}

function useTags(token:string) {
 const tags = useQuery({
  queryKey: ['topic-tags'],
  queryFn: async () => {
   const response = await apiFetch('/api/v1/topic-tags', token);
   if (!response.ok) throw new Error('读取话题失败');
   return (await response.json()) as TopicTagListResponse;
  },
  select: (data) => (data.items ?? []).map((tag) => tag.name),
 });
 return tags.data ?? [];
}
function TagChooser({values,selected,onChange}:{values:string[];selected:string[];onChange:(v:string[])=>void}){if(!values.length)return <Text style={s.privacy}>暂无可选标签</Text>;return <View style={s.tags}>{values.map(x=><TouchableOpacity key={x} onPress={()=>onChange(selected.includes(x)?selected.filter(t=>t!==x):[...selected,x])}><Text style={[s.chip,selected.includes(x)&&s.chipOn]}>{x}</Text></TouchableOpacity>)}</View>}
function PostInteractions({token,post,canInteract}:{token:string;post:Post;canInteract:boolean}){const qc=useQueryClient();const [liked,setLiked]=useState(post.liked_by_me);const [bookmarked,setBookmarked]=useState(post.bookmarked_by_me);const [likes,setLikes]=useState(post.like_count);const [open,setOpen]=useState(false);const [text,setText]=useState('');const [message,setMessage]=useState('');const comments=useQuery({queryKey:['post-comments',post.id],queryFn:async()=>{const r=await apiFetch(`/api/v1/posts/${post.id}/comments`,token);if(!r.ok)throw new Error('评论暂不可用');return (await r.json()) as CommentListResponse;},select:data=>data.items??[],enabled:open});const toggle=async(kind:'like'|'bookmark')=>{if(!canInteract)return;const current=kind==='like'?liked:bookmarked;const r=await apiFetch(`/api/v1/posts/${post.id}/${kind}`,token,{method:current?'DELETE':'PUT'});if(!r.ok){setMessage('操作失败，请稍后重试');return}if(kind==='like'){setLiked(!current);setLikes(n=>n+(current?-1:1))}else setBookmarked(!current);await Promise.all([qc.invalidateQueries({queryKey:['posts']}),qc.invalidateQueries({queryKey:['post-detail',post.id]}),qc.invalidateQueries({queryKey:['feed']}),qc.invalidateQueries({queryKey:[kind==='bookmark'?'my-bookmarks':'home']})])};const comment=async()=>{const r=await apiFetch(`/api/v1/posts/${post.id}/comments`,token,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({body:text})});const data=await r.json().catch(()=>({})) as {error?:string};if(!r.ok){setMessage(data.error||'评论失败');return}setText('');setMessage('');await qc.invalidateQueries({queryKey:['post-comments',post.id]})};return <View style={{marginTop:10}}><View style={s.scoreButtons}>{canInteract&&<TouchableOpacity onPress={()=>void toggle('like')}><Text style={[s.meta,liked&&{color:green,fontWeight:'800'}]}>{liked?'♥':'♡'} {likes}</Text></TouchableOpacity>}{canInteract&&<TouchableOpacity onPress={()=>void toggle('bookmark')}><Text style={[s.meta,bookmarked&&{color:green,fontWeight:'800'}]}>{bookmarked?'已收藏':'收藏'}</Text></TouchableOpacity>}<TouchableOpacity onPress={()=>setOpen(v=>!v)}><Text style={s.meta}>{open?'收起评论':'查看 / 评论'}</Text></TouchableOpacity></View>{!!message&&<Text style={s.privacy}>{message}</Text>}{open&&(comments.isLoading?<ActivityIndicator color={green}/>:comments.error?<Text style={s.privacy}>评论暂不可用</Text>:<View>{comments.data?.map(c=><Text key={c.id} style={s.privacy}>{c.author}{c.is_post_author?' · 楼主':''}：{c.body}</Text>)}{!comments.data?.length&&<Text style={s.privacy}>暂无评论</Text>}{canInteract&&<><TextInput value={text} onChangeText={setText} maxLength={1000} placeholder="写条评论（最多 1,000 字）" style={s.input}/><TouchableOpacity style={s.secondary} onPress={()=>void comment()}><Text style={s.report}>发表评论</Text></TouchableOpacity></>}</View>)}</View>}
export function ReportButton({ token, target, id }: { token: string; target: 'review' | 'post'; id: number }) {
 const [open,setOpen]=useState(false);
 const [detail,setDetail]=useState('');
 const [category,setCategory]=useState('other');
 const [sent,setSent]=useState(false);
 const [busy,setBusy]=useState(false);
 const reasons=[['spam','广告引流'],['academic_integrity','代写代考/泄题'],['copyright','版权侵权'],['personal_attack','人身攻击'],['teacher_claim','未经证实的教师指控'],['other','其他']] as const;

 const submit=async()=>{
  setBusy(true);
  try {
   const response=await apiFetch('/api/v1/reports',token,{
    method:'POST',
    headers:{'Content-Type':'application/json'},
    body:JSON.stringify({[target==='review'?'review_id':'post_id']:id,category,detail}),
   });
   if(!response.ok){
    const data=await response.json().catch(()=>({})) as {error?:string};
    throw new Error(data.error || '请稍后重试');
   }
   setSent(true);
   setOpen(false);
   Alert.alert('已提交','举报已进入运营复核队列。');
  } catch(error) {
   Alert.alert('无法举报',error instanceof Error ? error.message : '请稍后重试');
  } finally {
   setBusy(false);
  }
 };

 return <View style={{marginTop:8}}>
  {sent ? <Text style={s.privacy}>已举报</Text> : (
   <TouchableOpacity accessibilityRole="button" accessibilityState={{expanded:open}} disabled={busy}
    style={s.secondary} onPress={()=>setOpen(value=>!value)}>
    <Text style={s.report}>{open ? '收起举报表单' : '举报内容'}</Text>
   </TouchableOpacity>
  )}
  {open&&<View>
   <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={s.scoreButtons}>
    {reasons.map(([key,label])=><TouchableOpacity key={key} accessibilityRole="button"
     accessibilityState={{selected:category===key}} onPress={()=>setCategory(key)}>
     <Text style={[s.chip,category===key&&s.chipOn]}>{label}</Text>
    </TouchableOpacity>)}
   </ScrollView>
   <TextInput value={detail} onChangeText={setDetail} maxLength={900} placeholder="请补充举报原因（选填）" style={s.input}/>
   <TouchableOpacity disabled={busy} style={s.secondary} onPress={()=>void submit()}>
    <Text style={s.report}>{busy?'提交中…':'提交举报'}</Text>
   </TouchableOpacity>
  </View>}
 </View>;
}

const s=StyleSheet.create({screen:{flex:1,backgroundColor:'#f5f6f2'},header:{height:54,backgroundColor:'#fff',borderBottomWidth:1,borderBottomColor:'#e9ece6',flexDirection:'row',alignItems:'center',paddingHorizontal:20},back:{color:green,fontSize:15,fontWeight:'700'},headerTitle:{marginLeft:22,color:ink,fontSize:16,fontWeight:'800'},content:{padding:20,paddingBottom:40},code:{color:green,fontSize:12,fontWeight:'800'},title:{fontSize:24,fontWeight:'800',color:ink,marginTop:6},meta:{color:'#727d75',fontSize:12,lineHeight:19,marginTop:6},ratingCard:{backgroundColor:'#fff',borderRadius:16,padding:15,marginTop:18,borderWidth:1,borderColor:'#e9ece6'},section:{color:ink,fontWeight:'800',fontSize:15,marginBottom:7},tabs:{flexDirection:'row',gap:10,marginVertical:18},tab:{paddingVertical:9,paddingHorizontal:20,borderRadius:20,backgroundColor:'#fff'},tabOn:{backgroundColor:green},tabText:{color:'#647067',fontWeight:'700'},tabTextOn:{color:'#fff'},action:{backgroundColor:green,padding:13,borderRadius:12,alignItems:'center',marginVertical:10},actionText:{color:'#fff',fontWeight:'800'},notice:{color:green,fontSize:13,marginVertical:8},card:{backgroundColor:'#fff',borderRadius:15,padding:15,marginTop:11,borderWidth:1,borderColor:'#e9ece6'},cardTitle:{color:ink,fontSize:14,fontWeight:'800'},body:{color:'#4e5a52',fontSize:13,lineHeight:20,marginTop:9},empty:{color:'#7b857e',textAlign:'center',padding:24},hint:{color:'#9a5945',marginVertical:12},form:{backgroundColor:'#fff',padding:15,borderRadius:16,marginVertical:12},scoreRow:{marginVertical:6},scoreButtons:{flexDirection:'row',gap:8,alignItems:'center',marginVertical:8},score:{padding:7,color:'#829087',borderRadius:8,backgroundColor:'#f3f5f1',overflow:'hidden'},scoreOn:{color:'#fff',backgroundColor:green},input:{backgroundColor:'#f7f8f5',borderWidth:1,borderColor:'#e5e9e2',borderRadius:11,padding:12,color:ink,marginVertical:8,minHeight:46},privacy:{color:'#8c958e',fontSize:11,lineHeight:16},chip:{padding:8,borderRadius:10,overflow:'hidden',backgroundColor:'#f3f5f1',color:'#627067'},chipOn:{color:'#fff',backgroundColor:green},secondary:{paddingVertical:8},image:{width:'100%',height:190,borderRadius:12,marginTop:10},tags:{flexDirection:'row',flexWrap:'wrap',gap:8,marginVertical:8},report:{color:'#9a5945',fontSize:12,fontWeight:'700'}});
