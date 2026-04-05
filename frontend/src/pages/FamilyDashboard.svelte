<script>
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers, myLocation, tracking, focusUser } from '../lib/stores/map.js';
  import { mySosActive } from '../lib/stores/sos.js';
  import { connectivityStore } from '../lib/stores/connectivity.js';
  import { getUserColor, getUserColorLight } from '../lib/getUserColor.js';
  import { calculateDistance, formatDistance } from '../lib/tracking.js';
  import FamilyOrbit from '../components/primitives/FamilyOrbit.svelte';
  import GlobeCanvas from '../components/primitives/GlobeCanvas.svelte';
  import { clearHubBadge } from '../lib/stores/hubBadge.js';

  $: if (!$authUser) push('/login');

  const VIS_KEYS = {
    activity: 'kinnect_vis_activity', replay: 'kinnect_vis_replay',
    emergency: 'kinnect_vis_emergency', checkins: 'kinnect_vis_checkins',
  };
  let visited = { activity: true, replay: true, emergency: true, checkins: true };

  function visitFeature(key, route) {
    if (key) { localStorage.setItem(VIS_KEYS[key], '1'); visited = { ...visited, [key]: true }; }
    push(route);
  }

  let now = new Date();
  let clockInterval;
  onMount(() => {
    clockInterval = setInterval(() => { now = new Date(); }, 15000);
    clearHubBadge();
    visited = Object.fromEntries(Object.entries(VIS_KEYS).map(([k, v]) => [k, !!localStorage.getItem(v)]));
  });
  onDestroy(() => clearInterval(clockInterval));

  $: timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  $: dateStr = now.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' });

  function greeting() {
    const h = now.getHours();
    if (h < 5) return 'Up late'; if (h < 12) return 'Good morning';
    if (h < 17) return 'Good afternoon'; return 'Good evening';
  }
  $: firstName = ($authUser?.displayName || '').split(' ')[0] || 'there';

  $: members = Array.from($otherUsers.values());
  $: onlineCount = members.filter(m => m.online).length;
  $: movingCount = members.filter(m => m.online && m.speed > 1).length;
  $: sosMembers  = members.filter(m => m.sos?.active);
  $: allSafe     = sosMembers.length === 0 && !$mySosActive;

  function getInitials(n) { return (n||'').split(' ').map(s=>s[0]).join('').toUpperCase().slice(0,2)||'?'; }
  function presence(u) { if(u.sos?.active) return 'sos'; if(!u.online) return 'offline'; if(u.speed>1) return 'moving'; return 'online'; }
  function distText(u) { if(!u.lat||!u.lng||!$myLocation) return null; return formatDistance(calculateDistance($myLocation.latitude,$myLocation.longitude,u.lat,u.lng)); }
  function speedKmh(u) { return u.speed ? (u.speed*3.6).toFixed(0) : '0'; }

  const SR = 21; const SC = 2 * Math.PI * SR;
  $: safetyScore = allSafe ? (onlineCount > 0 ? Math.min(100, 55 + onlineCount * 8) : 55) : Math.max(10, 30 - sosMembers.length * 20);
  $: ringOffset = SC * (1 - safetyScore / 100);
  $: ringColor = allSafe ? '#10b981' : '#ef4444';

  const QUOTES = [
    { text: "Family is not an important thing. It's everything.", author: "Michael J. Fox" },
    { text: "The bond that links your true family is not blood, but respect and joy.", author: "Richard Bach" },
    { text: "In family life, love is the oil that eases friction.", author: "Friedrich Nietzsche" },
    { text: "Family means no one gets left behind or forgotten.", author: "Lilo & Stitch" },
    { text: "The family is the first essential cell of human society.", author: "Pope John XXIII" },
    { text: "A happy family is but an earlier heaven.", author: "George Bernard Shaw" },
    { text: "The love of a family is life's greatest blessing.", author: "Anonymous" },
    { text: "Other things may change us, but we start and end with family.", author: "Anthony Brandt" },
    { text: "Family is the anchor that holds us through life's storms.", author: "Anonymous" },
    { text: "Families are the compass that guides us.", author: "Brad Henry" },
    { text: "Having somewhere to go is home. Having someone to love is family.", author: "Anonymous" },
    { text: "The memories we make with our family is everything.", author: "Candace Cameron Bure" },
    { text: "You don't choose your family. They are God's gift to you, as you are to them.", author: "Desmond Tutu" },
    { text: "Family is not about blood. It's about who is willing to hold your hand when you need it the most.", author: "Anonymous" },
    { text: "Being a family means you are a part of something very wonderful.", author: "Lisa Weedn" },
    { text: "A family is a risky venture, because the greater the love, the greater the loss. That's the trade-off.", author: "Brad Pitt" },
    { text: "The strength of a family, like the strength of an army, lies in its loyalty to each other.", author: "Mario Puzo" },
    { text: "To us, family means putting your arms around each other and being there.", author: "Barbara Bush" },
    { text: "Family is the most important thing in the world.", author: "Princess Diana" },
    { text: "Home is where you are loved the most and act the worst.", author: "Marjorie Pay Hinckley" },
    { text: "When everything goes to hell, the people who stand by you without flinching — they are your family.", author: "Jim Butcher" },
    { text: "The informality of family life is a blessed condition that allows us to become our best while looking our worst.", author: "Marge Kennedy" },
    { text: "Rejoice with your family in the beautiful land of life.", author: "Albert Einstein" },
    { text: "Family and friends are hidden treasures. Seek them and enjoy their riches.", author: "Wanda Hope Carter" },
    { text: "A man travels the world over in search of what he needs, and returns home to find it.", author: "George Moore" },
    { text: "What can you do to promote world peace? Go home and love your family.", author: "Mother Teresa" },
    { text: "The only rock I know that stays steady is the rock that is the family.", author: "Nicholas Sparks" },
    { text: "There is no doubt that it is around the family and the home that all the greatest virtues are created.", author: "Winston Churchill" },
    { text: "Call it a clan, call it a network, call it a tribe, call it a family: whatever you call it, whoever you are, you need one.", author: "Jane Howard" },
    { text: "Families are like branches on a tree. We grow in different directions, yet our roots remain as one.", author: "Anonymous" },
    { text: "My family is my strength and my weakness.", author: "Aishwarya Rai" },
    { text: "Family makes you who you are and aren't.", author: "Anonymous" },
    { text: "At the end of the day, a loving family should find everything forgivable.", author: "Mark V. Olsen" },
    { text: "The greatest thing in family life is to take a hint when a hint is intended — and not to take a hint when a hint isn't intended.", author: "Robert Frost" },
    { text: "Family is a life jacket in the stormy sea of life.", author: "J.K. Rowling" },
    { text: "Friends are the family we choose for ourselves.", author: "Edna Buchanan" },
    { text: "A real friend is one who walks in when the rest of the world walks out.", author: "Walter Winchell" },
    { text: "Friendship is born at that moment when one person says to another, 'What! You too?'", author: "C.S. Lewis" },
    { text: "There is nothing on this earth more to be prized than true friendship.", author: "Thomas Aquinas" },
    { text: "A true friend is somebody who can make us do what we can.", author: "Ralph Waldo Emerson" },
    { text: "Friendship is the only cement that will ever hold the world together.", author: "Woodrow Wilson" },
    { text: "A friend is someone who knows all about you and still loves you.", author: "Elbert Hubbard" },
    { text: "True friendship comes when the silence between two people is comfortable.", author: "David Tyson" },
    { text: "Friends show their love in times of trouble, not in happiness.", author: "Euripides" },
    { text: "A sweet friendship refreshes the soul.", author: "Proverbs 27:9" },
    { text: "No friendship is an accident.", author: "O. Henry" },
    { text: "One loyal friend is worth ten thousand relatives.", author: "Euripides" },
    { text: "Walking with a friend in the dark is better than walking alone in the light.", author: "Helen Keller" },
    { text: "Life is nothing without friendship.", author: "Cicero" },
    { text: "The greatest gift of life is friendship, and I have received it.", author: "Hubert H. Humphrey" },
    { text: "A friend is one of the best things you can be and the greatest thing you can have.", author: "Sarah Valdez" },
    { text: "In the sweetness of friendship let there be laughter, for in the dew of little things the heart finds its morning and is refreshed.", author: "Khalil Gibran" },
    { text: "Friends are the siblings God never gave us.", author: "Mencius" },
    { text: "A friend may be waiting behind a stranger's face.", author: "Maya Angelou" },
    { text: "Good friends, good books, and a sleepy conscience: this is the ideal life.", author: "Mark Twain" },
    { text: "Lots of people want to ride with you in the limo, but what you want is someone who will take the bus with you when the limo breaks down.", author: "Oprah Winfrey" },
    { text: "True friends are like diamonds — bright, beautiful, valuable, and always in style.", author: "Nicole Richie" },
    { text: "Friendship is a sheltering tree.", author: "Samuel Taylor Coleridge" },
    { text: "There are no strangers here; only friends you haven't yet met.", author: "William Butler Yeats" },
    { text: "The only way to have a friend is to be one.", author: "Ralph Waldo Emerson" },
    { text: "Find a group of people who challenge and inspire you; spend a lot of time with them, and it will change your life.", author: "Amy Poehler" },
    { text: "Home is wherever I'm with my people.", author: "Anonymous" },
    { text: "Home is not a place — it's a feeling.", author: "Anonymous" },
    { text: "Where we love is home — home that our feet may leave, but not our hearts.", author: "Oliver Wendell Holmes" },
    { text: "There is nothing like staying at home for real comfort.", author: "Jane Austen" },
    { text: "The ache for home lives in all of us.", author: "Maya Angelou" },
    { text: "Home is the nicest word there is.", author: "Laura Ingalls Wilder" },
    { text: "Every house where love abides and friendship is a guest, is surely home.", author: "Henry Van Dyke" },
    { text: "You will never be completely at home again, because part of your heart will always be elsewhere.", author: "Miriam Adeney" },
    { text: "We may have all come on different ships, but we're in the same boat now.", author: "Martin Luther King Jr." },
    { text: "Wherever you go, go with all your heart.", author: "Confucius" },
    { text: "We are most alive when we're in love.", author: "John Updike" },
    { text: "Love is composed of a single soul inhabiting two bodies.", author: "Aristotle" },
    { text: "The best thing to hold onto in life is each other.", author: "Audrey Hepburn" },
    { text: "Being deeply loved by someone gives you strength, while loving someone deeply gives you courage.", author: "Lao Tzu" },
    { text: "There is only one happiness in this life, to love and be loved.", author: "George Sand" },
    { text: "You know you're in love when you can't fall asleep because reality is finally better than your dreams.", author: "Dr. Seuss" },
    { text: "To the world you may be one person, but to one person you may be the world.", author: "Bill Wilson" },
    { text: "Love recognizes no barriers.", author: "Maya Angelou" },
    { text: "The giving of love is an education in itself.", author: "Eleanor Roosevelt" },
    { text: "Love is a friendship set to music.", author: "Joseph Campbell" },
    { text: "Love does not dominate; it cultivates.", author: "Johann Wolfgang von Goethe" },
    { text: "Where there is love there is life.", author: "Mahatma Gandhi" },
    { text: "Life without love is like a tree without blossoms or fruit.", author: "Khalil Gibran" },
    { text: "We loved with a love that was more than love.", author: "Edgar Allan Poe" },
    { text: "I have decided to stick with love. Hate is too great a burden to bear.", author: "Martin Luther King Jr." },
    { text: "Believe you can and you're halfway there.", author: "Theodore Roosevelt" },
    { text: "The only impossible journey is the one you never begin.", author: "Tony Robbins" },
    { text: "It always seems impossible until it's done.", author: "Nelson Mandela" },
    { text: "Act as if what you do makes a difference. It does.", author: "William James" },
    { text: "What lies behind us and what lies before us are tiny matters compared to what lies within us.", author: "Ralph Waldo Emerson" },
    { text: "You are never too old to set another goal or to dream a new dream.", author: "C.S. Lewis" },
    { text: "The future belongs to those who believe in the beauty of their dreams.", author: "Eleanor Roosevelt" },
    { text: "In the middle of every difficulty lies opportunity.", author: "Albert Einstein" },
    { text: "Keep your face always toward the sunshine — and shadows will fall behind you.", author: "Walt Whitman" },
    { text: "It is during our darkest moments that we must focus to see the light.", author: "Aristotle" },
    { text: "The best time to plant a tree was 20 years ago. The second best time is now.", author: "Anonymous" },
    { text: "Everything you've ever wanted is on the other side of fear.", author: "George Addair" },
    { text: "You are braver than you believe, stronger than you seem, and smarter than you think.", author: "A.A. Milne" },
    { text: "Start where you are. Use what you have. Do what you can.", author: "Arthur Ashe" },
    { text: "Do what you can, with what you have, where you are.", author: "Theodore Roosevelt" },
    { text: "The secret of getting ahead is getting started.", author: "Mark Twain" },
    { text: "Don't watch the clock; do what it does. Keep going.", author: "Sam Levenson" },
    { text: "Hardships often prepare ordinary people for an extraordinary destiny.", author: "C.S. Lewis" },
    { text: "You don't have to be great to start, but you have to start to be great.", author: "Zig Ziglar" },
    { text: "With the new day comes new strength and new thoughts.", author: "Eleanor Roosevelt" },
    { text: "Courage is not the absence of fear, but rather the judgment that something else is more important than fear.", author: "Ambrose Redmoon" },
    { text: "Success is not final, failure is not fatal: it is the courage to continue that counts.", author: "Winston Churchill" },
    { text: "The only limit to our realization of tomorrow will be our doubts of today.", author: "Franklin D. Roosevelt" },
    { text: "You miss 100% of the shots you don't take.", author: "Wayne Gretzky" },
    { text: "Life is 10% what happens to us and 90% how we react to it.", author: "Charles R. Swindoll" },
    { text: "Happiness is not something ready made. It comes from your own actions.", author: "Dalai Lama" },
    { text: "What you get by achieving your goals is not as important as what you become by achieving your goals.", author: "Zig Ziglar" },
    { text: "I can't change the direction of the wind, but I can adjust my sails.", author: "Jimmy Dean" },
    { text: "It does not matter how slowly you go as long as you do not stop.", author: "Confucius" },
    { text: "When you reach the end of your rope, tie a knot in it and hang on.", author: "Franklin D. Roosevelt" },
    { text: "Safety is something that happens between your ears, not something you hold in your hands.", author: "Jeff Cooper" },
    { text: "The greatest protection any person can have is a loving family.", author: "Anonymous" },
    { text: "A safe family is a strong family.", author: "Anonymous" },
    { text: "Knowing your people are safe lets you sleep at night.", author: "Anonymous" },
    { text: "The best security blanket a child can have is parents who respect each other.", author: "Jane Blaustone" },
    { text: "Being safe is the foundation upon which everything else is built.", author: "Anonymous" },
    { text: "Caring for your people is the highest form of strength.", author: "Anonymous" },
    { text: "Peace of mind comes when you know your family is okay.", author: "Anonymous" },
    { text: "Gratitude turns what we have into enough.", author: "Aesop" },
    { text: "No act of kindness, no matter how small, is ever wasted.", author: "Aesop" },
    { text: "Enjoy the little things, for one day you may look back and realize they were the big things.", author: "Robert Brault" },
    { text: "The roots of all goodness lie in the soil of appreciation for goodness.", author: "Dalai Lama" },
    { text: "We can complain because rose bushes have thorns, or rejoice because thorn bushes have roses.", author: "Abraham Lincoln" },
    { text: "When I started counting my blessings, my whole life turned around.", author: "Willie Nelson" },
    { text: "Gratitude is not only the greatest of virtues, but the parent of all the others.", author: "Cicero" },
    { text: "Be thankful for what you have; you'll end up having more.", author: "Oprah Winfrey" },
    { text: "Appreciation is a wonderful thing. It makes what is excellent in others belong to us as well.", author: "Voltaire" },
    { text: "Kindness is a language which the deaf can hear and the blind can see.", author: "Mark Twain" },
    { text: "A single act of kindness throws out roots in all directions, and the roots spring up and make new trees.", author: "Amelia Earhart" },
    { text: "Carry out a random act of kindness with no expectation of reward.", author: "Princess Diana" },
    { text: "The purpose of our lives is to be happy.", author: "Dalai Lama" },
    { text: "Life is really simple, but we insist on making it complicated.", author: "Confucius" },
    { text: "In three words I can sum up everything I've learned about life: it goes on.", author: "Robert Frost" },
    { text: "Life is what happens when you're busy making other plans.", author: "John Lennon" },
    { text: "The biggest adventure you can take is to live the life of your dreams.", author: "Oprah Winfrey" },
    { text: "Do not dwell in the past, do not dream of the future, concentrate the mind on the present moment.", author: "Buddha" },
    { text: "Not how long, but how well you have lived is the main thing.", author: "Seneca" },
    { text: "Life is either a daring adventure or nothing at all.", author: "Helen Keller" },
    { text: "The unexamined life is not worth living.", author: "Socrates" },
    { text: "Turn your wounds into wisdom.", author: "Oprah Winfrey" },
    { text: "The best and most beautiful things in the world cannot be seen or even touched — they must be felt with the heart.", author: "Helen Keller" },
    { text: "Life isn't about finding yourself. Life is about creating yourself.", author: "George Bernard Shaw" },
    { text: "Spread love everywhere you go. Let no one ever come to you without leaving happier.", author: "Mother Teresa" },
    { text: "We make a living by what we get, but we make a life by what we give.", author: "Winston Churchill" },
    { text: "The best preparation for tomorrow is doing your best today.", author: "H. Jackson Brown Jr." },
    { text: "Be yourself; everyone else is already taken.", author: "Oscar Wilde" },
    { text: "If you want to lift yourself up, lift up someone else.", author: "Booker T. Washington" },
    { text: "Strive not to be a success, but rather to be of value.", author: "Albert Einstein" },
    { text: "Twenty years from now you will be more disappointed by the things that you didn't do than by the ones you did do.", author: "Mark Twain" },
    { text: "The way to get started is to quit talking and begin doing.", author: "Walt Disney" },
    { text: "The human spirit is stronger than anything that can happen to it.", author: "C.C. Scott" },
    { text: "Tough times never last, but tough people do.", author: "Robert H. Schuller" },
    { text: "Rock bottom became the solid foundation on which I rebuilt my life.", author: "J.K. Rowling" },
    { text: "A smooth sea never made a skilled sailor.", author: "Franklin D. Roosevelt" },
    { text: "Fall seven times, stand up eight.", author: "Japanese Proverb" },
    { text: "Strength does not come from winning. Your struggles develop your strengths.", author: "Arnold Schwarzenegger" },
    { text: "The oak fought the wind and was broken, the willow bent when it must and survived.", author: "Robert Jordan" },
    { text: "Stars can't shine without darkness.", author: "Anonymous" },
    { text: "She stood in the storm, and when the wind did not blow her way, she adjusted her sails.", author: "Elizabeth Edwards" },
    { text: "You were given this life because you are strong enough to live it.", author: "Anonymous" },
    { text: "Out of difficulties grow miracles.", author: "Jean de La Bruyere" },
    { text: "Alone we can do so little; together we can do so much.", author: "Helen Keller" },
    { text: "If you want to go fast, go alone. If you want to go far, go together.", author: "African Proverb" },
    { text: "No man is an island, entire of itself.", author: "John Donne" },
    { text: "We rise by lifting others.", author: "Robert Ingersoll" },
    { text: "The greatness of a community is most accurately measured by the compassionate actions of its members.", author: "Coretta Scott King" },
    { text: "It takes a village to raise a child.", author: "African Proverb" },
    { text: "Coming together is a beginning, staying together is progress, and working together is success.", author: "Henry Ford" },
    { text: "Unity is strength. When there is teamwork and collaboration, wonderful things can be achieved.", author: "Mattie Stepanek" },
    { text: "None of us is as smart as all of us.", author: "Ken Blanchard" },
    { text: "A candle loses nothing by lighting another candle.", author: "James Keller" },
    { text: "Happiness is not by chance, but by choice.", author: "Jim Rohn" },
    { text: "The most wasted of days is one without laughter.", author: "E.E. Cummings" },
    { text: "Think of all the beauty still left around you and be happy.", author: "Anne Frank" },
    { text: "For every minute you are angry you lose sixty seconds of happiness.", author: "Ralph Waldo Emerson" },
    { text: "The sun himself is weak when he first rises, and gathers strength and courage as the day gets on.", author: "Charles Dickens" },
    { text: "Count your age by friends, not years. Count your life by smiles, not tears.", author: "John Lennon" },
    { text: "Very little is needed to make a happy life; it is all within yourself, in your way of thinking.", author: "Marcus Aurelius" },
    { text: "Happiness often sneaks in through a door you didn't know you left open.", author: "John Barrymore" },
    { text: "There is nothing either good or bad, but thinking makes it so.", author: "William Shakespeare" },
    { text: "Today is a good day to have a good day.", author: "Anonymous" },
    { text: "Let us always meet each other with smile, for the smile is the beginning of love.", author: "Mother Teresa" },
    { text: "Be the reason someone smiles today.", author: "Anonymous" },
    { text: "Happiness is a warm puppy.", author: "Charles M. Schulz" },
    { text: "The most important thing is to enjoy your life — to be happy — it's all that matters.", author: "Audrey Hepburn" },
  ];
  let quoteIdx = Math.floor(Math.random() * QUOTES.length);
  let quoteVisible = true;
  let quoteInterval;

  let mounted = false;
  function cycleQuote() {
    quoteVisible = false;
    setTimeout(() => { quoteIdx = (quoteIdx + 1) % QUOTES.length; quoteVisible = true; }, 400);
  }

  // Mouse glow tracking
  let mouseX = 0, mouseY = 0;
  function handleMouseMove(e) { mouseX = e.clientX; mouseY = e.clientY; mouseOnDash = true; }
  let mouseOnDash = false;

  // Responsive globe size — fills available left area
  let globeSize = 400;
  function updateGlobeSize() {
    if (typeof window === 'undefined') return;
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    if (vw < 768) {
      // Mobile: globe inline in content, fits without dominating the viewport
      globeSize = Math.round(Math.min(260, Math.max(200, vw * 0.62)));
    } else {
      const sidebarW = vw >= 1200 ? Math.min(460, vw * 0.35) : Math.min(420, vw * 0.38);
      const leftW = Math.max(400, vw - sidebarW);
      globeSize = Math.round(Math.min(520, Math.max(320, Math.min(vh * 0.54, leftW * 0.52))));
    }
  }

  onMount(() => {
    requestAnimationFrame(() => { mounted = true; });
    quoteInterval = setInterval(cycleQuote, 10000);
    updateGlobeSize();
    window.addEventListener('resize', updateGlobeSize);
  });
  onDestroy(() => {
    clearInterval(clockInterval);
    clearInterval(quoteInterval);
    window.removeEventListener('resize', updateGlobeSize);
  });
</script>

<div class="d" class:d-ready={mounted}
  on:mousemove={handleMouseMove}
  on:mouseleave={() => mouseOnDash = false}>
  <div class="d-aurora" aria-hidden="true"></div>
  <div class="d-noise" aria-hidden="true"></div>

  <!-- Mouse cursor glow — follows pointer, desktop only -->
  <div class="d-cursor-glow" class:d-cursor-visible={mouseOnDash}
    style="left:{mouseX}px;top:{mouseY}px" aria-hidden="true"></div>

  <!-- FULL-SCREEN ORBIT BACKGROUND -->
  <div class="d-orbit-bg" aria-hidden="true">
    <FamilyOrbit />
  </div>

  <!-- LEFT COLUMN — desktop: centered globe + info + quote -->
  <div class="d-left-hud" aria-hidden="false">
    <div class="d-globe-col">

      <!-- TOP: eyebrow + live status pills -->
      <div class="d-globe-top">
        <div class="d-globe-eyebrow">
          <span class="d-globe-blip"></span>
          FAMILY NETWORK
        </div>
        <div class="d-globe-status-row">
          <span class="d-gsr-pill" class:gsr-on={$tracking}>
            <span class="d-gsr-dot"></span>
            Tracking {$tracking ? 'ON' : 'OFF'}
          </span>
          <span class="d-gsr-pill" class:gsr-on={onlineCount > 0}>
            <span class="d-gsr-dot"></span>
            {onlineCount} online
          </span>
          <span class="d-gsr-pill" class:gsr-safe={allSafe} class:gsr-alert={!allSafe}>
            <span class="d-gsr-dot"></span>
            {allSafe ? 'All safe' : `${sosMembers.length + ($mySosActive ? 1 : 0)} alert`}
          </span>
          <span class="d-gsr-pill" class:gsr-on={$connectivityStore.isOnline} class:gsr-warn={!$connectivityStore.isOnline}>
            <span class="d-gsr-dot"></span>
            {$connectivityStore.isOnline ? 'Connected' : 'Offline'}
          </span>
        </div>
      </div>

      <!-- CENTER: Globe with flanking stat cards on large screens -->
      <div class="d-globe-center">
        <!-- Left: Network card -->
        <div class="d-globe-side d-globe-side-l">
          <div class="d-side-card">
            <div class="d-side-label">Network</div>
            <div class="d-side-big">{members.length}</div>
            <div class="d-side-sub">Members</div>
            <div class="d-side-divider"></div>
            <div class="d-side-row">
              <span class="d-sd dot-on"></span><span>{onlineCount} online</span>
            </div>
            <div class="d-side-row">
              <span class="d-sd dot-mv"></span><span>{movingCount} moving</span>
            </div>
            <div class="d-side-row">
              <span class="d-sd" style="background:rgba(255,255,255,0.12)"></span>
              <span>{members.length - onlineCount} offline</span>
            </div>
          </div>
        </div>

        <GlobeCanvas size={globeSize} />

        <!-- Right: Safety card -->
        <div class="d-globe-side d-globe-side-r">
          <div class="d-side-card">
            <div class="d-side-label">Safety</div>
            <div class="d-side-big" style="color:{ringColor}">{safetyScore}</div>
            <div class="d-side-sub">Score</div>
            <div class="d-side-divider"></div>
            <div class="d-side-row" class:row-safe={allSafe} class:row-sos={!allSafe}>
              <span class="d-sd" style="background:{ringColor}"></span>
              <span>{allSafe ? 'All safe' : `${sosMembers.length} SOS`}</span>
            </div>
            <div class="d-side-row">
              <span class="d-sd" style="background:{$connectivityStore.isOnline ? '#10b981' : '#f59e0b'}"></span>
              <span>{$connectivityStore.isOnline ? 'Connected' : 'Offline'}</span>
            </div>
            <div class="d-side-row">
              <span class="d-sd" style="background:{$tracking ? '#3b82f6' : 'rgba(255,255,255,0.12)'}"></span>
              <span>Tracking {$tracking ? 'on' : 'off'}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- BOTTOM: member ring + location + quote -->
      <div class="d-globe-bottom">

        <!-- Interactive member avatars — click to focus on map -->
        {#if members.length > 0}
          <div class="d-member-ring">
            {#each members.slice(0, 7) as user (user.userId)}
              {@const color = getUserColor(user.userId)}
              {@const pres = presence(user)}
              <!-- svelte-ignore a11y-no-static-element-interactions -->
              <!-- svelte-ignore a11y-click-events-have-key-events -->
              <div class="d-mr-bubble" style="--mc:{color}"
                on:click={() => { focusUser.set(user.userId); push('/'); }}
                title="{user.displayName} — {pres}">
                <span class="d-mr-init">{getInitials(user.displayName)}</span>
                <span class="d-mr-dot"
                  class:dot-on={pres==='online'} class:dot-mv={pres==='moving'}
                  class:dot-sos={pres==='sos'} class:dot-off={pres==='offline'}></span>
              </div>
            {/each}
            {#if members.length > 7}
              <div class="d-mr-more">+{members.length - 7}</div>
            {/if}
          </div>
        {/if}

        {#if $myLocation?.latitude != null}
          <div class="d-loc-info">
            <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/></svg>
            <span class="d-loc-you">You are here</span>
            <span class="d-loc-coords">
              {Math.abs($myLocation.latitude).toFixed(3)}°{$myLocation.latitude >= 0 ? 'N' : 'S'}
              &nbsp;·&nbsp;
              {Math.abs($myLocation.longitude).toFixed(3)}°{$myLocation.longitude >= 0 ? 'E' : 'W'}
            </span>
          </div>
        {:else}
          <div class="d-loc-info d-loc-unknown">
            <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
            Location not shared
          </div>
        {/if}

        <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <div class="d-quote-globe" class:dqg-on={quoteVisible} role="complementary"
          on:click={cycleQuote} title="Click for next quote">
          <span class="d-qg-mark" aria-hidden="true">"</span>
          <p class="d-qg-text">{QUOTES[quoteIdx].text}"</p>
          <span class="d-qg-author">— {QUOTES[quoteIdx].author}</span>
          <span class="d-qg-cycle" aria-hidden="true">↻ next</span>
        </div>
      </div>

    </div>
  </div>

  <!-- HEADER — glass, floating -->
  <header class="d-header">
    <button class="d-back" on:click={() => push('/')} aria-label="Back to map">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
      Map
    </button>
    <div class="d-clock">{timeStr}</div>
  </header>

  <!-- CONTENT OVERLAY — scrollable panels floating over the orbit -->
  <div class="d-content">

    <!-- Hero: greeting + safety -->
    <section class="d-hero">
      <div class="d-greet">
        <h1 class="d-name">
          <span class="d-greet-word">{greeting()},&nbsp;</span><span class="d-name-word">{firstName}</span>
        </h1>
        <p class="d-date">{dateStr}</p>
      </div>
      <div class="d-safety" aria-label="Safety score {safetyScore}">
        <svg width="56" height="56" viewBox="0 0 56 56" aria-hidden="true">
          <circle cx="28" cy="28" r="{SR}" stroke="rgba(255,255,255,0.06)" stroke-width="4" fill="none"/>
          <circle cx="28" cy="28" r="{SR}" stroke="{ringColor}" stroke-width="4" fill="none"
            stroke-linecap="round" stroke-dasharray="{SC}" stroke-dashoffset="{ringOffset}"
            transform="rotate(-90 28 28)" class="d-ring-progress"/>
        </svg>
        <span class="d-safety-score">{safetyScore}</span>
        <div class="d-safety-badge" class:badge-safe={allSafe} class:badge-sos={!allSafe}>
          <span class="d-badge-dot"></span>
          {allSafe ? 'All safe' : `${sosMembers.length + ($mySosActive ? 1 : 0)} SOS`}
        </div>
      </div>
    </section>

    <!-- Mobile Globe — inline in scroll flow, hidden on desktop (desktop uses d-left-hud) -->
    <div class="d-mobile-globe">
      <GlobeCanvas size={globeSize} />
      {#if $myLocation?.latitude != null}
        <div class="d-mob-coords">
          <span class="d-mob-you">You are here</span>
          <span class="d-mob-ll">
            {Math.abs($myLocation.latitude).toFixed(3)}°{$myLocation.latitude >= 0 ? 'N' : 'S'}
            &nbsp;·&nbsp;
            {Math.abs($myLocation.longitude).toFixed(3)}°{$myLocation.longitude >= 0 ? 'E' : 'W'}
          </span>
        </div>
      {/if}
    </div>

    <!-- Quote (centered, over orbit) -->
    <div class="d-quote" class:quote-on={quoteVisible} aria-live="polite">
      <p class="d-quote-text">{QUOTES[quoteIdx].text}</p>
      <span class="d-quote-author">— {QUOTES[quoteIdx].author}</span>
    </div>

    <!-- Stats row -->
    <section class="d-stats">
      <div class="d-stat">
        <span class="d-stat-val">{onlineCount}</span>
        <span class="d-stat-lbl">Online</span>
      </div>
      <div class="d-stat" class:stat-active={$tracking}>
        <span class="d-stat-val">{$tracking ? 'ON' : 'OFF'}</span>
        <span class="d-stat-lbl">Tracking</span>
        {#if $tracking}<span class="d-stat-dot" aria-hidden="true"></span>{/if}
      </div>
      <div class="d-stat">
        <span class="d-stat-val">{movingCount}</span>
        <span class="d-stat-lbl">Moving</span>
      </div>
      <div class="d-stat" class:stat-warn={!$connectivityStore.isOnline}>
        <span class="d-stat-val">{$connectivityStore.isOnline ? 'OK' : 'OFF'}</span>
        <span class="d-stat-lbl">Network</span>
      </div>
    </section>

    <!-- Network -->
    <section class="d-panel d-panel-network">
      <header class="d-panel-head">
        <h2>Your Network</h2>
        <span class="d-badge">{members.length}</span>
      </header>
      {#if members.length === 0}
        <div class="d-empty">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          <p>No one in your network yet</p>
          <button class="d-cta" on:click={() => push('/')}>Open Map</button>
        </div>
      {:else}
        <div class="d-members">
          {#each members as user (user.userId)}
            {@const color = getUserColor(user.userId)}
            {@const pres = presence(user)}
            {@const dist = distText(user)}
            <button class="d-member" class:m-sos={pres==='sos'} class:m-off={pres==='offline'}
              style="--mc:{color}" on:click={() => { focusUser.set(user.userId); push('/'); }}>
              <div class="m-av">
                <span class="m-init">{getInitials(user.displayName)}</span>
                <span class="m-dot" class:dot-sos={pres==='sos'} class:dot-off={pres==='offline'} class:dot-mv={pres==='moving'} class:dot-on={pres==='online'}></span>
              </div>
              <div class="m-info">
                <span class="m-name">{user.displayName || 'Unknown'}</span>
                <span class="m-status" class:m-sos-text={pres==='sos'}>
                  {pres==='sos'?'SOS':pres==='moving'?speedKmh(user)+' km/h':pres==='offline'?'Offline':'Online'}
                </span>
              </div>
              {#if dist}<span class="m-dist">{dist}</span>{/if}
            </button>
          {/each}
        </div>
      {/if}
    </section>

    <!-- Quick Actions -->
    <section class="d-panel d-panel-actions">
      <header class="d-panel-head"><h2>Quick Actions</h2></header>
      <div class="d-actions">
        <button class="d-act act-map" on:click={() => visitFeature(null, '/')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 6v16l7-4 8 4 7-4V2l-7 4-8-4-7 4z"/><line x1="8" y1="2" x2="8" y2="18"/><line x1="16" y1="6" x2="16" y2="22"/></svg>
          <span>Live Map</span>
        </button>
        <button class="d-act act-activity" on:click={() => visitFeature('activity', '/activity')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
          <span>Activity</span>
          {#if !visited.activity}<span class="d-dot"></span>{/if}
        </button>
        <button class="d-act act-replay" on:click={() => visitFeature('replay', '/replay')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 .49-4.61"/></svg>
          <span>Routes</span>
          {#if !visited.replay}<span class="d-dot"></span>{/if}
        </button>
        <button class="d-act act-sos" class:act-sos-on={$mySosActive} on:click={() => visitFeature('emergency', '/emergency')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
          <span>Emergency</span>
          {#if !visited.emergency}<span class="d-dot d-dot-red"></span>{/if}
        </button>
        <button class="d-act act-checkin" on:click={() => visitFeature('checkins', '/checkins')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
          <span>Check-ins</span>
          {#if !visited.checkins}<span class="d-dot d-dot-cyan"></span>{/if}
        </button>
        <button class="d-act act-network" on:click={() => visitFeature(null, '/')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="5" r="2"/><circle cx="5" cy="19" r="2"/><circle cx="19" cy="19" r="2"/><line x1="12" y1="7" x2="12" y2="11"/><line x1="8.5" y1="16.5" x2="12" y2="11"/><line x1="15.5" y1="16.5" x2="12" y2="11"/></svg>
          <span>Network</span>
        </button>
      </div>
    </section>

    <div class="d-spacer" style="height:calc(var(--safe-bottom,0px) + 28px)"></div>
  </div>
</div>

<style>
  :root { --ease-expo: cubic-bezier(0.16,1,0.3,1); }

  /* ═══ Shell ═════════════════════════════════════════════════════════ */
  .d {
    height: 100dvh;
    background: var(--surface-0, #050812);
    color: #fff;
    font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
    position: relative;
    overflow: hidden;
    opacity: 0; transition: opacity 0.5s ease;
  }
  .d.d-ready { opacity: 1; }

  /* ═══ Aurora + noise (behind everything) ════════════════════════════ */
  .d-aurora {
    position: fixed; inset: 0; pointer-events: none; z-index: 0;
    background:
      radial-gradient(ellipse 70% 50% at 15% 20%, rgba(99,102,241,0.12) 0%, transparent 65%),
      radial-gradient(ellipse 60% 45% at 85% 75%, rgba(139,92,246,0.08) 0%, transparent 60%);
  }
  .d-noise {
    position: fixed; inset: 0; pointer-events: none; z-index: 0; opacity: 0.025;
    background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
    background-size: 180px; mix-blend-mode: overlay;
  }

  /* ═══ ORBIT — FULL SCREEN BACKGROUND ═══════════════════════════════ */
  .d-orbit-bg {
    position: absolute;
    inset: 0;
    z-index: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: auto;
  }
  /* Orbit is hidden on all viewports — the GlobeCanvas is the primary 3D visual on
     both desktop (left HUD) and mobile (inline in scroll flow). Hiding saves RAF CPU. */
  .d-orbit-bg { display: none; }

  /* ═══ HEADER — floating glass bar ══════════════════════════════════ */
  .d-header {
    position: fixed; top: 0; left: 0; right: 0; z-index: 20;
    display: flex; align-items: center; justify-content: space-between;
    padding: calc(var(--safe-top, 0px) + 10px) 20px 10px;
    background: rgba(5,8,18,0.55);
    backdrop-filter: blur(24px) saturate(1.5);
    -webkit-backdrop-filter: blur(24px) saturate(1.5);
    border-bottom: 1px solid rgba(255,255,255,0.04);
  }
  .d-back {
    display: flex; align-items: center; gap: 4px;
    background: rgba(255,255,255,0.06); border: 1px solid rgba(255,255,255,0.08);
    border-radius: 20px; padding: 5px 11px 5px 7px;
    color: rgba(255,255,255,0.65); font-size: 12px; font-weight: 600;
    cursor: pointer; transition: background 0.15s, color 0.15s;
  }
  .d-back:hover { background: rgba(255,255,255,0.10); color: #fff; }
  .d-back:active { transform: scale(0.96); }
  .d-clock {
    font-size: 20px; font-weight: 700; letter-spacing: -0.03em;
    color: rgba(255,255,255,0.85); font-variant-numeric: tabular-nums;
    font-family: var(--font-display, system-ui);
  }

  /* ═══ CONTENT OVERLAY — scrolls over the orbit ═════════════════════ */
  .d-content {
    position: relative; z-index: 5;
    height: 100dvh;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    overscroll-behavior-y: contain;
    padding-top: calc(var(--safe-top, 0px) + 44px);
    pointer-events: none; /* let clicks through to orbit */
  }
  /* Re-enable pointer events on actual content */
  .d-content > * { pointer-events: auto; }

  /* On mobile: content flows normally (orbit behind, content on top) */
  /* On desktop: no-scroll sidebar — everything fits in 100dvh */
  @media (min-width: 768px) {
    .d-content {
      position: absolute;
      top: 0; right: 0; bottom: 0;
      width: min(420px, 38vw);
      padding: calc(var(--safe-top, 0px) + 52px) 16px calc(var(--safe-bottom, 0px) + 12px) 16px;
      background: linear-gradient(90deg, transparent 0%, rgba(5,8,18,0.6) 30%, rgba(5,8,18,0.88) 100%);
      backdrop-filter: blur(12px);
      -webkit-backdrop-filter: blur(12px);
      /* KEY: no outer scroll — flex distributes sections */
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }
    /* Hero + stats never shrink */
    .d-hero { flex-shrink: 0; }
    .d-stats { flex-shrink: 0; padding-top: 8px; }
    /* Network panel grows to fill remaining space */
    .d-panel-network {
      flex: 1;
      min-height: 0;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
    .d-panel-network .d-members {
      flex: 1;
      min-height: 0;
      overflow-y: auto;
      scrollbar-width: thin;
      scrollbar-color: rgba(255,255,255,0.06) transparent;
    }
    .d-panel-network .d-empty {
      flex: 1; min-height: 0;
      display: flex; flex-direction: column; justify-content: center;
    }
    /* Actions panel never shrinks */
    .d-panel-actions { flex-shrink: 0; }
    /* Hide mobile-only spacer */
    .d-spacer { display: none; }
    /* Compact hero on desktop */
    .d-hero { padding: 10px 16px 0; gap: 8px; }
    .d-name { font-size: clamp(1.4rem, 3.5vw, 1.8rem) !important; }
    .d-panel { margin: 8px 0 0; padding: 10px 12px 10px; }
    .d-safety svg { width: 48px; height: 48px; }
    .d-safety-score { font-size: 11px !important; top: 14px !important; }
  }
  @media (min-width: 1200px) {
    .d-content { width: min(460px, 35vw); }
  }

  /* ═══ Stagger ══════════════════════════════════════════════════════ */
  .d-hero, .d-stats, .d-panel {
    opacity: 0; transform: translateY(10px);
    transition: opacity 0.5s var(--ease-expo), transform 0.5s var(--ease-expo);
  }
  .d-ready .d-hero   { opacity: 1; transform: none; transition-delay: 0.05s; }
  .d-ready .d-stats   { opacity: 1; transform: none; transition-delay: 0.15s; }
  .d-ready .d-panel   { opacity: 1; transform: none; transition-delay: 0.22s; }
  .d-ready .d-panel + .d-panel { transition-delay: 0.30s; }
  /* .d-quote controlled purely by .quote-on — no stagger override */
  .d-quote { opacity: 0; transition: opacity 0.6s ease; }
  .d-quote.quote-on { opacity: 1; }


  /* ═══ Hero ═════════════════════════════════════════════════════════ */
  .d-hero {
    display: flex; align-items: flex-start; justify-content: space-between;
    padding: 16px 20px 0; gap: 12px;
  }
  .d-greet { flex: 1; min-width: 0; }
  .d-name {
    font-size: clamp(1.7rem, 5vw, 2.4rem); font-weight: 400;
    letter-spacing: -0.03em; line-height: 1.15; margin: 0;
    font-family: var(--font-display, system-ui);
    color: rgba(255,255,255,0.55); /* base: muted gray for "Up late," */
  }
  .d-greet-word {
    /* "Up late," — medium weight, muted */
    font-weight: 400;
    color: rgba(255,255,255,0.55);
  }
  .d-name-word {
    /* "Ankur" — bold, white→purple gradient */
    font-weight: 800;
    background: linear-gradient(135deg, #fff 30%, #c4b5fd 100%);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
  }
  .d-date { margin: 5px 0 0; font-size: 12px; font-weight: 400; color: rgba(255,255,255,0.28); letter-spacing: 0.01em; }

  .d-safety { position: relative; display: flex; flex-direction: column; align-items: center; gap: 3px; flex-shrink: 0; }
  .d-ring-progress { transition: stroke-dashoffset 1s cubic-bezier(0.4,0,0.2,1), stroke 0.4s; }
  .d-safety-score { position: absolute; top: 16px; left: 50%; transform: translateX(-50%); font-size: 13px; font-weight: 800; color: rgba(255,255,255,0.9); font-family: var(--font-display, system-ui); }
  .d-safety-badge { display: flex; align-items: center; gap: 4px; font-size: 9px; font-weight: 700; padding: 2px 7px; border-radius: 20px; white-space: nowrap; }
  .badge-safe { background: rgba(16,185,129,0.12); border: 1px solid rgba(16,185,129,0.25); color: #10b981; }
  .badge-sos  { background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.25); color: #f87171; }
  .d-badge-dot { width: 4px; height: 4px; border-radius: 50%; background: currentColor; animation: pulse 2s ease-in-out infinite; }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.3} }

  /* ═══ Quote ════════════════════════════════════════════════════════ */
  .d-quote {
    padding: 8px 20px; text-align: center;
    opacity: 0; transition: opacity 0.6s ease;
  }
  .d-quote.quote-on { opacity: 1; }
  .d-quote-text { font-size: 11.5px; font-style: italic; font-family: Georgia, serif; color: rgba(255,255,255,0.30); line-height: 1.6; margin: 0 0 3px; }
  .d-quote-author { font-size: 8px; font-weight: 700; color: rgba(139,92,246,0.4); text-transform: uppercase; letter-spacing: 0.06em; }
  @media (min-width: 768px) {
    .d-quote { display: none; } /* Replaced by .d-quote-left in the left HUD on desktop */
  }

  /* ═══ Stats ════════════════════════════════════════════════════════ */
  .d-stats { display: flex; gap: 6px; padding: 12px 20px 0; }
  .d-stat {
    flex: 1;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.07);
    border-radius: 12px; padding: 8px 6px;
    display: flex; flex-direction: column; gap: 1px; align-items: center;
    position: relative; overflow: hidden;
    backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
    transition: border-color 0.2s, background 0.2s;
  }
  .d-stat:active { background: rgba(255,255,255,0.07); }
  .d-stat-val { font-size: 18px; font-weight: 800; color: #fff; line-height: 1; letter-spacing: -0.04em; font-variant-numeric: tabular-nums; font-family: var(--font-display, system-ui); }
  .d-stat-lbl { font-size: 8px; font-weight: 600; color: rgba(255,255,255,0.25); text-transform: uppercase; letter-spacing: 0.07em; }
  .stat-active { border-color: rgba(16,185,129,0.25); }
  .stat-active .d-stat-val { color: #10b981; }
  .stat-warn { border-color: rgba(245,158,11,0.25); }
  .stat-warn .d-stat-val { color: #f59e0b; }
  .d-stat-dot { position: absolute; top: 5px; right: 5px; width: 4px; height: 4px; border-radius: 50%; background: #10b981; animation: pulse 2s ease-in-out infinite; }

  /* ═══ Panel (glass card for Network & Actions) ═════════════════════ */
  .d-panel {
    margin: 12px 20px 0;
    background: rgba(255,255,255,0.03);
    border: 1px solid rgba(255,255,255,0.06);
    border-radius: 16px;
    padding: 14px 14px 12px;
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }
  .d-panel-head { display: flex; align-items: center; gap: 6px; margin-bottom: 10px; }
  .d-panel-head h2 { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: rgba(255,255,255,0.35); margin: 0; flex: 1; }
  .d-badge { font-size: 9px; font-weight: 700; background: rgba(99,102,241,0.12); border: 1px solid rgba(99,102,241,0.2); color: rgba(139,92,246,0.8); padding: 1px 6px; border-radius: 20px; }

  /* ═══ Members — list layout (works in sidebar) ═════════════════════ */
  .d-members { display: flex; flex-direction: column; gap: 6px; }
  .d-member {
    display: flex; align-items: center; gap: 10px;
    background: rgba(255,255,255,0.03);
    border: 1px solid rgba(255,255,255,0.05);
    border-radius: 12px; padding: 8px 10px;
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s, transform 0.1s;
    -webkit-tap-highlight-color: transparent;
  }
  .d-member:hover { border-color: rgba(255,255,255,0.12); background: rgba(255,255,255,0.05); }
  .d-member:active { transform: scale(0.98); }
  .d-member.m-sos { border-color: rgba(239,68,68,0.25); }
  .d-member.m-off { opacity: 0.45; }

  .m-av { position: relative; width: 32px; height: 32px; border-radius: 50%; background: color-mix(in srgb, var(--mc,#6366f1) 15%, transparent); border: 2px solid var(--mc,#6366f1); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
  .m-init { font-size: 11px; font-weight: 800; color: var(--mc,#6366f1); line-height: 1; user-select: none; }
  .m-dot { position: absolute; bottom: -1px; right: -1px; width: 9px; height: 9px; border-radius: 50%; border: 2px solid var(--surface-0, #050812); }
  .dot-on { background: #10b981; } .dot-mv { background: #3b82f6; } .dot-off { background: #475569; } .dot-sos { background: #ef4444; }
  .m-info { flex: 1; min-width: 0; }
  .m-name { display: block; font-size: 12px; font-weight: 700; color: #fff; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .m-status { font-size: 10px; color: rgba(255,255,255,0.35); }
  .m-sos-text { color: #f87171; font-weight: 700; }
  .m-dist { font-size: 9px; color: rgba(255,255,255,0.22); font-variant-numeric: tabular-nums; flex-shrink: 0; }

  /* ═══ Empty ════════════════════════════════════════════════════════ */
  .d-empty {
    text-align: center; display: flex; flex-direction: column; align-items: center; gap: 8px;
    color: rgba(255,255,255,0.28); font-size: 12px; padding: 16px 0;
  }
  .d-cta {
    background: linear-gradient(135deg, var(--primary-500, #4f46e5), var(--primary-600, #7c3aed));
    color: #fff; border: none; border-radius: 10px; padding: 7px 14px;
    font-size: 11px; font-weight: 700; cursor: pointer;
    transition: transform 0.12s, box-shadow 0.2s;
    box-shadow: 0 2px 10px rgba(99,102,241,0.3);
  }
  .d-cta:hover { transform: translateY(-1px); } .d-cta:active { transform: scale(0.97); }

  /* ═══ Actions ══════════════════════════════════════════════════════ */
  .d-actions { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; }
  .d-act {
    position: relative;
    background: rgba(255,255,255,0.03); border: 1px solid rgba(255,255,255,0.06);
    border-radius: 12px; padding: 12px 6px 10px;
    display: flex; flex-direction: column; align-items: center; gap: 6px;
    cursor: pointer; color: rgba(255,255,255,0.65); font-size: 10px; font-weight: 600;
    transition: border-color 0.2s, transform 0.1s, background 0.2s;
    -webkit-tap-highlight-color: transparent;
  }
  .d-act:hover { border-color: rgba(255,255,255,0.12); background: rgba(255,255,255,0.05); }
  .d-act:active { transform: scale(0.95); }
  .act-map { color: #818cf8; } .act-activity { color: #34d399; } .act-replay { color: #fbbf24; }
  .act-sos { color: #f87171; } .act-checkin { color: #22d3ee; } .act-network { color: #a78bfa; }
  .act-sos-on { border-color: rgba(239,68,68,0.25); animation: sos-b 2s ease-in-out infinite; }
  @keyframes sos-b { 0%,100%{border-color:rgba(239,68,68,0.15)} 50%{border-color:rgba(239,68,68,0.45)} }

  .d-dot { position: absolute; top: 6px; right: 6px; width: 6px; height: 6px; border-radius: 50%; background: #f59e0b; box-shadow: 0 0 5px rgba(245,158,11,0.6); }
  .d-dot-red { background: #ef4444; box-shadow: 0 0 5px rgba(239,68,68,0.6); }
  .d-dot-cyan { background: #22d3ee; box-shadow: 0 0 5px rgba(34,211,238,0.6); }

  /* ═══ Mobile Globe section ══════════════════════════════════════════ */
  .d-mobile-globe {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 4px 20px 4px;
  }
  @media (min-width: 768px) {
    .d-mobile-globe { display: none; }
  }
  .d-mob-coords {
    display: flex; flex-direction: column; align-items: center; gap: 2px;
  }
  .d-mob-you {
    font-size: 8px; font-weight: 800;
    color: rgba(167,139,250,0.55);
    text-transform: uppercase; letter-spacing: 0.12em;
  }
  .d-mob-ll {
    font-size: 10px; font-weight: 600;
    font-family: var(--font-mono, monospace);
    color: rgba(255,255,255,0.22);
    letter-spacing: 0.03em;
    font-variant-numeric: tabular-nums;
  }

  /* ═══ Reduced motion ═══════════════════════════════════════════════ */
  @media (prefers-reduced-motion: reduce) {
    .d-aurora, .d-badge-dot, .d-stat-dot, .d-member.m-sos, .d-act.act-sos-on { animation: none !important; }
    .d-hero, .d-stats, .d-panel, .d-quote { opacity: 1 !important; transform: none !important; transition: none !important; }
  }

  /* ═══ Mouse cursor glow ════════════════════════════════════════════ */
  .d-cursor-glow {
    position: fixed;
    width: 360px; height: 360px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(99,102,241,0.055) 0%, rgba(139,92,246,0.025) 45%, transparent 70%);
    transform: translate(-50%, -50%);
    pointer-events: none;
    z-index: 2;
    opacity: 0;
    transition: opacity 0.4s ease;
    display: none;
  }
  @media (min-width: 768px) {
    .d-cursor-glow { display: block; }
    .d-cursor-glow.d-cursor-visible { opacity: 1; }
  }

  /* ═══ Left HUD overlay — globe fills the left area ════════════════ */
  .d-left-hud {
    display: none;
  }
  @media (min-width: 768px) {
    .d-left-hud {
      display: flex;
      align-items: stretch;   /* stretch so globe-col fills full height */
      justify-content: center;
      position: absolute;
      top: 0; bottom: 0; left: 0;
      right: min(420px, 38vw);
      pointer-events: none;
      z-index: 3;
    }
    .d-left-hud > :global(*) { pointer-events: auto; }
  }
  @media (min-width: 1200px) {
    .d-left-hud { right: min(460px, 35vw); }
  }

  /* ── Globe column layout ────────────────────────────────────────── */
  .d-globe-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    padding: calc(var(--safe-top, 0px) + 64px) 24px 32px;
    max-width: 480px;
    width: 100%;
    height: 100%;
    pointer-events: auto;
  }

  /* ── Top section (eyebrow + status pills) ─────────────────────── */
  .d-globe-top {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    width: 100%;
  }

  /* ── Status pills row ─────────────────────────────────────────── */
  .d-globe-status-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    justify-content: center;
  }
  .d-gsr-pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 4px 10px 4px 8px;
    border-radius: 20px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.07);
    font-size: 9px;
    font-weight: 700;
    color: rgba(255,255,255,0.35);
    letter-spacing: 0.04em;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    transition: border-color 0.2s, color 0.2s;
  }
  .d-gsr-dot {
    width: 5px; height: 5px;
    border-radius: 50%;
    background: rgba(255,255,255,0.18);
    flex-shrink: 0;
    transition: background 0.2s, box-shadow 0.2s;
  }
  /* ON / green */
  .gsr-on {
    border-color: rgba(16,185,129,0.22);
    color: rgba(52,211,153,0.75);
  }
  .gsr-on .d-gsr-dot {
    background: #10b981;
    box-shadow: 0 0 5px rgba(16,185,129,0.55);
    animation: blip-g 2.5s ease-in-out infinite;
  }
  /* SAFE / green */
  .gsr-safe {
    border-color: rgba(16,185,129,0.22);
    color: rgba(52,211,153,0.75);
  }
  .gsr-safe .d-gsr-dot {
    background: #10b981;
    box-shadow: 0 0 5px rgba(16,185,129,0.55);
  }
  /* ALERT / red */
  .gsr-alert {
    border-color: rgba(239,68,68,0.28);
    color: rgba(248,113,113,0.85);
  }
  .gsr-alert .d-gsr-dot {
    background: #ef4444;
    box-shadow: 0 0 5px rgba(239,68,68,0.60);
    animation: pulse 1.5s ease-in-out infinite;
  }
  /* WARN / amber */
  .gsr-warn {
    border-color: rgba(245,158,11,0.24);
    color: rgba(251,191,36,0.75);
  }
  .gsr-warn .d-gsr-dot {
    background: #f59e0b;
    box-shadow: 0 0 5px rgba(245,158,11,0.50);
  }

  /* ── Bottom section (loc + quote) ────────────────────────────── */
  .d-globe-bottom {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    width: 100%;
  }

  .d-globe-eyebrow {
    display: flex; align-items: center; gap: 7px;
    font-size: 8.5px; font-weight: 800;
    color: rgba(255,255,255,0.14);
    letter-spacing: 0.15em; text-transform: uppercase;
    font-family: var(--font-mono, monospace);
  }
  .d-globe-blip {
    width: 5px; height: 5px; border-radius: 50%;
    background: #10b981;
    box-shadow: 0 0 6px rgba(16,185,129,0.7);
    animation: blip-g 2.5s ease-in-out infinite;
    flex-shrink: 0;
  }
  @keyframes blip-g { 0%,100%{opacity:1} 50%{opacity:0.2} }

  .d-loc-info {
    display: flex; align-items: center; gap: 6px; flex-wrap: wrap; justify-content: center;
    color: rgba(167,139,250,0.55);
  }
  .d-loc-you {
    font-size: 9px; font-weight: 800;
    color: rgba(167,139,250,0.65);
    text-transform: uppercase; letter-spacing: 0.10em;
  }
  .d-loc-coords {
    font-size: 11px; font-weight: 600;
    font-family: var(--font-mono, monospace);
    color: rgba(255,255,255,0.30);
    letter-spacing: 0.03em;
    font-variant-numeric: tabular-nums;
  }
  .d-loc-unknown {
    font-size: 9px; font-weight: 600;
    color: rgba(255,255,255,0.18);
    letter-spacing: 0.06em; text-transform: uppercase;
    gap: 5px;
  }

  /* ── Quote in globe column ──────────────────────────────────────── */
  .d-quote-globe {
    position: relative;
    width: 100%; max-width: 340px;
    padding: 16px 20px 14px;
    background: rgba(5,8,18,0.50);
    border: 1px solid rgba(255,255,255,0.06);
    border-top: 1px solid rgba(139,92,246,0.12);
    border-radius: 12px;
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    cursor: pointer;
    text-align: center;
    opacity: 0; transform: translateY(8px);
    transition: opacity 0.55s ease, transform 0.55s ease, border-color 0.2s, background 0.2s;
  }
  .d-quote-globe.dqg-on { opacity: 1; transform: translateY(0); }
  .d-quote-globe:hover {
    background: rgba(99,102,241,0.06);
    border-color: rgba(99,102,241,0.18);
  }
  .d-qg-mark {
    display: block;
    font-size: 36px; font-family: Georgia, serif;
    color: rgba(139,92,246,0.14);
    line-height: 1; margin-bottom: -8px;
    user-select: none; pointer-events: none;
  }
  .d-qg-text {
    font-size: 12px; font-style: italic;
    font-family: Georgia, 'Times New Roman', serif;
    color: rgba(255,255,255,0.38);
    line-height: 1.7; margin: 0 0 8px;
  }
  .d-qg-author {
    display: block;
    font-size: 9px; font-weight: 700;
    color: rgba(139,92,246,0.50);
    text-transform: uppercase; letter-spacing: 0.08em;
    margin-bottom: 8px;
  }
  .d-qg-cycle {
    display: block; font-size: 8px; font-weight: 600;
    color: rgba(255,255,255,0.10); letter-spacing: 0.05em;
    transition: color 0.2s;
  }
  .d-quote-globe:hover .d-qg-cycle { color: rgba(139,92,246,0.40); }

  /* ═══ Globe center — globe + flanking side cards ════════════════════ */
  .d-globe-center {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    gap: 14px;
  }

  /* Side cards hidden below 960px (not enough room beside globe) */
  .d-globe-side { display: none; flex: 1; max-width: 155px; }
  @media (min-width: 960px) {
    .d-globe-side { display: flex; align-items: center; justify-content: center; }
  }

  .d-side-card {
    width: 100%;
    background: rgba(5,8,18,0.55);
    border: 1px solid rgba(255,255,255,0.07);
    border-radius: 16px;
    padding: 14px 12px 12px;
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    text-align: center;
    cursor: default;
    transition: transform 0.35s cubic-bezier(0.34,1.56,0.64,1), border-color 0.2s, box-shadow 0.2s;
    transform-style: preserve-3d;
  }
  .d-globe-side-l .d-side-card:hover {
    transform: perspective(500px) rotateY(6deg) translateY(-3px);
    border-color: rgba(139,92,246,0.20);
    box-shadow: 4px 8px 28px rgba(0,0,0,0.22);
  }
  .d-globe-side-r .d-side-card:hover {
    transform: perspective(500px) rotateY(-6deg) translateY(-3px);
    border-color: rgba(139,92,246,0.20);
    box-shadow: -4px 8px 28px rgba(0,0,0,0.22);
  }

  .d-side-label {
    font-size: 8px; font-weight: 800; text-transform: uppercase;
    letter-spacing: 0.14em; color: rgba(255,255,255,0.20);
    margin-bottom: 6px;
    font-family: var(--font-mono, monospace);
  }
  .d-side-big {
    font-size: 38px; font-weight: 800;
    color: rgba(255,255,255,0.90);
    line-height: 1; letter-spacing: -0.05em;
    font-family: var(--font-display, system-ui);
    font-variant-numeric: tabular-nums;
    transition: color 0.4s;
  }
  .d-side-sub {
    font-size: 8px; font-weight: 600;
    color: rgba(255,255,255,0.20); text-transform: uppercase;
    letter-spacing: 0.08em; margin-top: 2px;
  }
  .d-side-divider {
    height: 1px; background: rgba(255,255,255,0.06); margin: 10px 0 8px;
  }
  .d-side-row {
    display: flex; align-items: center; gap: 6px;
    font-size: 9px; font-weight: 600; color: rgba(255,255,255,0.28);
    margin-top: 5px; text-align: left;
  }
  .row-safe { color: rgba(52,211,153,0.75); }
  .row-sos  { color: rgba(248,113,113,0.80); }
  .d-sd {
    width: 5px; height: 5px; border-radius: 50%; flex-shrink: 0;
    background: rgba(255,255,255,0.15);
  }
  .d-sd.dot-on { background: #10b981; box-shadow: 0 0 4px rgba(16,185,129,0.55); }
  .d-sd.dot-mv { background: #3b82f6; box-shadow: 0 0 4px rgba(59,130,246,0.55); }

  /* ═══ Interactive member avatar ring ════════════════════════════════ */
  .d-member-ring {
    display: flex; align-items: center; gap: 8px;
    flex-wrap: wrap; justify-content: center;
  }
  .d-mr-bubble {
    position: relative; width: 40px; height: 40px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--mc,#6366f1) 14%, rgba(5,8,18,0.75));
    border: 2px solid color-mix(in srgb, var(--mc,#6366f1) 60%, transparent);
    display: flex; align-items: center; justify-content: center;
    cursor: pointer;
    transition: transform 0.25s cubic-bezier(0.34,1.56,0.64,1), box-shadow 0.25s, border-color 0.2s;
    -webkit-tap-highlight-color: transparent;
    user-select: none;
  }
  .d-mr-bubble:hover {
    transform: scale(1.18) translateY(-3px);
    border-color: var(--mc, #6366f1);
    box-shadow: 0 6px 20px color-mix(in srgb, var(--mc,#6366f1) 40%, transparent);
  }
  .d-mr-bubble:active { transform: scale(0.92); }
  .d-mr-init {
    font-size: 12px; font-weight: 800;
    color: color-mix(in srgb, var(--mc,#6366f1) 90%, white);
    line-height: 1; user-select: none; pointer-events: none;
  }
  .d-mr-dot {
    position: absolute; bottom: -1px; right: -1px;
    width: 11px; height: 11px;
    border-radius: 50%; border: 2px solid rgba(5,8,18,0.9);
    background: rgba(255,255,255,0.15);
  }
  .d-mr-dot.dot-on  { background: #10b981; }
  .d-mr-dot.dot-mv  { background: #3b82f6; }
  .d-mr-dot.dot-sos { background: #ef4444; box-shadow: 0 0 5px rgba(239,68,68,0.7); }
  .d-mr-dot.dot-off { background: #475569; }
  .d-mr-more {
    width: 40px; height: 40px; border-radius: 50%;
    background: rgba(255,255,255,0.04);
    border: 2px solid rgba(255,255,255,0.10);
    display: flex; align-items: center; justify-content: center;
    font-size: 9px; font-weight: 700; color: rgba(255,255,255,0.30);
  }
</style>
